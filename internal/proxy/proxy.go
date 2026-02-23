package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elazarl/goproxy"

	"proxy-tui/internal/model"
	"proxy-tui/pkg/ca"
)

// FlowProvider gives read-only access to flow data and server metadata.
// Use this when a consumer only needs to observe flows (e.g. IPC Server).
type FlowProvider interface {
	FlowStore() *FlowStore
	Port() int
	BindAddress() string
}

// FlowSource is the full interface used by the ViewModel.
// It extends FlowProvider with event streaming and mutable proxy configuration.
type FlowSource interface {
	FlowProvider
	Events() <-chan model.FlowEvent
	SSLProxyList() *SSLProxyList
	MapRules() *model.MapRuleStore
}

// Proxy represents the HTTP/HTTPS proxy server
type Proxy struct {
	server       *http.Server
	goproxy      *goproxy.ProxyHttpServer
	flowStore    *FlowStore
	eventChan    chan model.FlowEvent
	ca           *ca.CA
	port         int
	bindAddress  string
	running      bool
	mapRules     *model.MapRuleStore
	sslProxyList *SSLProxyList
}

// Config holds proxy configuration
type Config struct {
	Port          int
	BindAddress   string
	CA            *ca.CA
	MaxFlows      int
	Verbose       bool
	EventChanSize int
}

// DefaultConfig returns default proxy configuration
func DefaultConfig() *Config {
	return &Config{
		Port:          9090,
		BindAddress:   "0.0.0.0",
		MaxFlows:      10000,
		Verbose:       false,
		EventChanSize: 1000,
	}
}

// New creates a new proxy instance
func New(config *Config) (*Proxy, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Load or generate CA
	certificate := config.CA
	if certificate == nil {
		var err error
		certificate, err = ca.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load CA: %w", err)
		}
	}

	eventChan := make(chan model.FlowEvent, config.EventChanSize)
	flowStore := NewFlowStore(config.MaxFlows, eventChan)

	goproxyServer := goproxy.NewProxyHttpServer()
	goproxyServer.Verbose = config.Verbose

	bindAddr := config.BindAddress
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	sslProxyList := NewSSLProxyList()

	p := &Proxy{
		goproxy:      goproxyServer,
		flowStore:    flowStore,
		eventChan:    eventChan,
		ca:           certificate,
		port:         config.Port,
		bindAddress:  bindAddr,
		mapRules:     model.NewMapRuleStore(),
		sslProxyList: sslProxyList,
	}

	// Setup MITM with conditional interception based on SSL proxy list
	mitmConfig := DefaultMITMConfig(certificate)
	SetupConditionalMITMWithCallback(goproxyServer, mitmConfig, sslProxyList, p.handleTunnel)

	// Setup request handler
	goproxyServer.OnRequest().DoFunc(p.handleRequest)

	// Setup response handler
	goproxyServer.OnResponse().DoFunc(p.handleResponse)

	return p, nil
}

// handleTunnel is called when a CONNECT tunnel is established
func (p *Proxy) handleTunnel(host string, isMITM bool) {
	// Only record if this is a passthrough tunnel (not MITM)
	// MITM connections will be recorded via handleRequest
	if isMITM {
		return
	}

	// Create a flow for the tunneled connection
	flow := &model.Flow{
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Tunneled:  true,
		Request: &model.Request{
			Method: "CONNECT",
			URL:    "https://" + host,
			Host:   host,
			Path:   "",
			Proto:  "HTTP/1.1",
		},
	}

	// Store flow
	p.flowStore.Add(flow)
}

// handleRequest processes incoming requests
func (p *Proxy) handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	startTime := time.Now()

	// Read request body
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Construct full URL (for HTTPS MITM, req.URL only has the path)
	fullURL := req.URL.String()
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		scheme := "http"
		if req.TLS != nil || ctx.Req.URL.Scheme == "https" {
			scheme = "https"
		}
		fullURL = fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.RequestURI())
	}

	// Create flow
	flow := &model.Flow{
		StartTime: startTime,
		Request: &model.Request{
			Method:  req.Method,
			URL:     fullURL,
			Host:    req.Host,
			Path:    req.URL.Path,
			Proto:   req.Proto,
			Headers: req.Header.Clone(),
			Body:    bodyBytes,
		},
	}

	// Store flow
	flowID := p.flowStore.Add(flow)

	// Store flow ID in context for response handler
	ctx.UserData = flowID

	// Check for mapping rules (skip when paused — full bypass)
	if !p.flowStore.IsPaused() {
		if rule := p.mapRules.FindMatch(fullURL); rule != nil {
			var resp *http.Response

			switch rule.Type {
			case model.MapLocal:
				resp = p.serveLocalFile(rule, req, flow)
			case model.MapRemote:
				resp = p.fetchRemote(rule, req, flow)
			}

			if resp != nil {
				// Mark flow as mapped
				switch rule.Type {
				case model.MapLocal:
					flow.Mapped = "local"
				case model.MapRemote:
					flow.Mapped = "remote"
				}
				// Update flow with the mapped response
				flow.Response = &model.Response{
					StatusCode: resp.StatusCode,
					Status:     resp.Status,
					Proto:      resp.Proto,
					Headers:    resp.Header.Clone(),
				}
				if resp.Body != nil {
					bodyBytes, _ := io.ReadAll(resp.Body)
					flow.Response.Body = bodyBytes
					resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
				flow.EndTime = time.Now()
				p.flowStore.Update(flow, model.FlowEventResponse)
				return req, resp
			}
		}
	}

	return req, nil
}

// handleResponse processes responses
func (p *Proxy) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	flowID, ok := ctx.UserData.(model.FlowID)
	if !ok || resp == nil {
		return resp
	}

	flow := p.flowStore.Get(flowID)
	if flow == nil {
		return resp
	}

	// Read response body
	var bodyBytes []byte
	if resp.Body != nil {
		bodyBytes, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Update flow with response
	flow.Response = &model.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		Headers:    resp.Header.Clone(),
		Body:       bodyBytes,
	}
	flow.EndTime = time.Now()

	// Update store and send event
	p.flowStore.Update(flow, model.FlowEventResponse)

	return resp
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	addr := fmt.Sprintf("%s:%d", p.bindAddress, p.port)
	p.server = &http.Server{
		Addr:    addr,
		Handler: p.goproxy,
	}

	p.running = true
	return p.server.ListenAndServe()
}

// StartAsync starts the proxy server in a goroutine
func (p *Proxy) StartAsync() error {
	addr := fmt.Sprintf("%s:%d", p.bindAddress, p.port)
	p.server = &http.Server{
		Addr:    addr,
		Handler: p.goproxy,
	}

	p.running = true
	go func() {
		if err := p.server.ListenAndServe(); err != http.ErrServerClosed {
			// Log error but don't crash
		}
	}()
	return nil
}

// BindAddress returns the bind address
func (p *Proxy) BindAddress() string {
	return p.bindAddress
}

// Stop stops the proxy server
func (p *Proxy) Stop(ctx context.Context) error {
	p.running = false
	if p.server != nil {
		return p.server.Shutdown(ctx)
	}
	return nil
}

// Events returns the event channel
func (p *Proxy) Events() <-chan model.FlowEvent {
	return p.eventChan
}

// FlowStore returns the flow store
func (p *Proxy) FlowStore() *FlowStore {
	return p.flowStore
}

// CA returns the CA
func (p *Proxy) CA() *ca.CA {
	return p.ca
}

// Port returns the proxy port
func (p *Proxy) Port() int {
	return p.port
}

// IsRunning returns whether the proxy is running
func (p *Proxy) IsRunning() bool {
	return p.running
}

// MapRules returns the map rule store
func (p *Proxy) MapRules() *model.MapRuleStore {
	return p.mapRules
}

// SSLProxyList returns the SSL proxy list
func (p *Proxy) SSLProxyList() *SSLProxyList {
	return p.sslProxyList
}
