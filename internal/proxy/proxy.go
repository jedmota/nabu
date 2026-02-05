package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"

	"proxy-tui/internal/model"
	"proxy-tui/pkg/ca"
)

// SSLProxyList manages the list of hosts to perform MITM on
type SSLProxyList struct {
	patterns []string
	mu       sync.RWMutex
}

// NewSSLProxyList creates a new SSL proxy list
func NewSSLProxyList() *SSLProxyList {
	return &SSLProxyList{
		patterns: make([]string, 0),
	}
}

// Add adds a pattern to the list
func (s *SSLProxyList) Add(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check if already exists
	for _, p := range s.patterns {
		if p == pattern {
			return
		}
	}
	s.patterns = append(s.patterns, pattern)
}

// Remove removes a pattern from the list
func (s *SSLProxyList) Remove(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.patterns {
		if p == pattern {
			s.patterns = append(s.patterns[:i], s.patterns[i+1:]...)
			return
		}
	}
}

// Clear removes all patterns
func (s *SSLProxyList) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patterns = make([]string, 0)
}

// Patterns returns a copy of all patterns
func (s *SSLProxyList) Patterns() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.patterns))
	copy(result, s.patterns)
	return result
}

// Match checks if a host matches any pattern in the list
func (s *SSLProxyList) Match(host string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If list is empty, don't match anything (passthrough all)
	if len(s.patterns) == 0 {
		return false
	}

	for _, pattern := range s.patterns {
		if matchHostPattern(host, pattern) {
			return true
		}
	}
	return false
}

// matchHostPattern checks if host matches a glob-like pattern
func matchHostPattern(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard matching
	if pattern == "*" {
		return true
	}

	// *.example.com matches example.com and sub.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		domain := pattern[2:] // "example.com"
		return host == domain || strings.HasSuffix(host, suffix)
	}

	// General glob pattern with * (e.g., *json.com, api.*, *google*)
	if strings.Contains(pattern, "*") {
		return matchGlob(host, pattern)
	}

	// Try regex if it looks like one
	if strings.ContainsAny(pattern, "^$()[]{}|+?\\") {
		re, err := regexp.Compile(pattern)
		if err == nil {
			return re.MatchString(host)
		}
	}

	return false
}

// matchGlob matches a string against a glob pattern with * wildcards
func matchGlob(s, pattern string) bool {
	// Convert glob to regex: escape special chars, replace * with .*
	regexPattern := "^"
	for _, c := range pattern {
		switch c {
		case '*':
			regexPattern += ".*"
		case '.', '+', '?', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			regexPattern += "\\" + string(c)
		default:
			regexPattern += string(c)
		}
	}
	regexPattern += "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
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

	// Check for mapping rules
	if rule := p.mapRules.FindMatch(fullURL); rule != nil {
		var resp *http.Response

		switch rule.Type {
		case model.MapLocal:
			resp = p.serveLocalFile(rule, req, flow)
		case model.MapRemote:
			resp = p.fetchRemote(rule, req, flow)
		}

		if resp != nil {
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

	return req, nil
}

// serveLocalFile creates an HTTP response from a local file
func (p *Proxy) serveLocalFile(rule *model.MapRule, req *http.Request, flow *model.Flow) *http.Response {
	localPath := rule.Replacement

	// Expand home directory
	if strings.HasPrefix(localPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			localPath = home + localPath[1:]
		}
	}

	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		// Return 404 if file not found
		return &http.Response{
			StatusCode: 404,
			Status:     "404 Not Found",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type":   []string{"text/plain"},
				"X-Mapped-Local": []string{"error: " + err.Error()},
			},
			Body:          io.NopCloser(bytes.NewBufferString("File not found: " + localPath)),
			ContentLength: int64(len("File not found: " + localPath)),
			Request:       req,
		}
	}

	// Check if it's a JSONC file (our format)
	if strings.HasSuffix(strings.ToLower(localPath), ".jsonc") {
		resp := parseJSONCResponseFile(content, req, localPath)
		if resp != nil {
			return resp
		}
		// JSONC file but failed to parse - return error
		errMsg := "Failed to parse JSONC response file"
		return &http.Response{
			StatusCode: 500,
			Status:     "500 Internal Server Error",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type":   []string{"text/plain"},
				"X-Mapped-Local": []string{localPath},
			},
			Body:          io.NopCloser(bytes.NewBufferString(errMsg)),
			ContentLength: int64(len(errMsg)),
			Request:       req,
		}
	}

	// Try to parse as old HTTP response format
	if resp := parseOldHTTPFormat(content, req, localPath); resp != nil {
		return resp
	}

	// Fallback: treat as raw body
	statusCode := rule.GetStatusCode()
	contentType := rule.GetContentType()

	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Content-Type":   []string{contentType},
			"Content-Length": []string{fmt.Sprintf("%d", len(content))},
			"X-Mapped-Local": []string{localPath},
		},
		Body:          io.NopCloser(bytes.NewBuffer(content)),
		ContentLength: int64(len(content)),
		Request:       req,
	}
}

// fetchRemote fetches from a remote URL and returns the response as if from original
func (p *Proxy) fetchRemote(rule *model.MapRule, originalReq *http.Request, flow *model.Flow) *http.Response {
	// Parse replacement as the new origin (scheme + host + port)
	replacementURL, err := url.Parse(rule.Replacement)
	if err != nil {
		return &http.Response{
			StatusCode: 502,
			Status:     "502 Bad Gateway",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type":    []string{"text/plain"},
				"X-Mapped-Remote": []string{"error: invalid replacement URL"},
			},
			Body:          io.NopCloser(bytes.NewBufferString("Invalid replacement URL: " + rule.Replacement)),
			ContentLength: int64(len("Invalid replacement URL: " + rule.Replacement)),
			Request:       originalReq,
		}
	}

	// Build the new URL: replacement origin + original path + query
	remoteURL := replacementURL.Scheme + "://" + replacementURL.Host + originalReq.URL.Path
	if originalReq.URL.RawQuery != "" {
		remoteURL += "?" + originalReq.URL.RawQuery
	}

	// Create a new request to the remote URL
	newReq, err := http.NewRequest(originalReq.Method, remoteURL, nil)
	if err != nil {
		return &http.Response{
			StatusCode: 502,
			Status:     "502 Bad Gateway",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type":    []string{"text/plain"},
				"X-Mapped-Remote": []string{"error: " + err.Error()},
			},
			Body:          io.NopCloser(bytes.NewBufferString("Failed to create request: " + err.Error())),
			ContentLength: int64(len("Failed to create request: " + err.Error())),
			Request:       originalReq,
		}
	}

	// Copy headers from original request
	for key, values := range originalReq.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}

	// Copy body if present
	if originalReq.Body != nil {
		bodyBytes, err := io.ReadAll(originalReq.Body)
		if err == nil && len(bodyBytes) > 0 {
			newReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			newReq.ContentLength = int64(len(bodyBytes))
			// Restore original request body for potential retry
			originalReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	// Use a client that doesn't follow redirects (we want to return redirects as-is)
	// Explicitly configure Transport to handle HTTP/HTTPS correctly
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DisableKeepAlives: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Execute the request
	resp, err := client.Do(newReq)
	if err != nil {
		return &http.Response{
			StatusCode: 502,
			Status:     "502 Bad Gateway",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type":    []string{"text/plain"},
				"X-Mapped-Remote": []string{"error: " + err.Error()},
			},
			Body:          io.NopCloser(bytes.NewBufferString("Failed to fetch remote: " + err.Error())),
			ContentLength: int64(len("Failed to fetch remote: " + err.Error())),
			Request:       originalReq,
		}
	}

	// Rewrite any Location header in redirect responses to use original host
	if location := resp.Header.Get("Location"); location != "" {
		// If the Location points to the replacement host, rewrite it to the original
		if strings.HasPrefix(location, replacementURL.Scheme+"://"+replacementURL.Host) {
			// Replace the scheme and host with the original
			originalScheme := "https"
			if !strings.HasPrefix(flow.Request.URL, "https://") {
				originalScheme = "http"
			}
			newLocation := strings.Replace(location,
				replacementURL.Scheme+"://"+replacementURL.Host,
				originalScheme+"://"+originalReq.Host, 1)
			resp.Header.Set("Location", newLocation)
		}
	}

	// Set the request back to original for proper flow tracking
	resp.Request = originalReq

	return resp
}

// MapLocalResponse represents the JSONC file structure for mapped responses
type MapLocalResponse struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

// parseJSONCResponseFile parses a JSONC response file
func parseJSONCResponseFile(content []byte, req *http.Request, localPath string) *http.Response {
	// Strip JSONC comments
	jsonContent := stripJSONComments(content)

	// Parse as JSONC format
	var resp MapLocalResponse
	if err := json.Unmarshal(jsonContent, &resp); err != nil {
		return nil
	}

	if resp.Status == 0 {
		resp.Status = 200
	}

	return buildResponseFromJSON(&resp, req, localPath)
}

// stripJSONComments removes // comments from JSONC
func stripJSONComments(data []byte) []byte {
	var result strings.Builder
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		// Find // comment (not inside string)
		inString := false
		commentIdx := -1
		for i := 0; i < len(line); i++ {
			if line[i] == '"' && (i == 0 || line[i-1] != '\\') {
				inString = !inString
			}
			if !inString && i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
				commentIdx = i
				break
			}
		}
		if commentIdx >= 0 {
			line = line[:commentIdx]
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	return []byte(result.String())
}

// buildResponseFromJSON creates an HTTP response from parsed JSONC
func buildResponseFromJSON(resp *MapLocalResponse, req *http.Request, localPath string) *http.Response {
	// Build body
	var body []byte
	if resp.Body == nil {
		body = []byte{}
	} else {
		switch v := resp.Body.(type) {
		case string:
			body = []byte(v)
		case []byte:
			body = v
		case map[string]interface{}:
			// JSON object - marshal it
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		case []interface{}:
			// JSON array - marshal it
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		default:
			// Try to marshal whatever it is
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		}
	}

	// Build headers from the parsed headers map
	headers := make(http.Header)
	if resp.Headers != nil {
		for key, value := range resp.Headers {
			headers.Set(key, value)
		}
	}

	// Set Content-Length
	headers.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	headers.Set("X-Mapped-Local", localPath)

	statusText := resp.StatusText
	if statusText == "" {
		statusText = http.StatusText(resp.Status)
	}

	return &http.Response{
		StatusCode:    resp.Status,
		Status:        fmt.Sprintf("%d %s", resp.Status, statusText),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        headers,
		Body:          io.NopCloser(bytes.NewBuffer(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

// parseOldHTTPFormat parses the old HTTP response format for backward compatibility
func parseOldHTTPFormat(content []byte, req *http.Request, localPath string) *http.Response {
	// Find the separator between headers and body (empty line)
	headerEnd := bytes.Index(content, []byte("\n\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(content, []byte("\r\n\r\n"))
		if headerEnd == -1 {
			return nil
		}
		headerEnd += 2 // account for \r\n
	}

	headerSection := string(content[:headerEnd])
	body := content[headerEnd+2:] // skip the empty line

	lines := strings.Split(headerSection, "\n")
	if len(lines) == 0 {
		return nil
	}

	// Parse status line: HTTP/1.1 200 OK
	statusLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(statusLine, "HTTP/") {
		return nil
	}

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return nil
	}

	proto := parts[0]
	statusCode := 200
	statusText := "OK"
	fmt.Sscanf(parts[1], "%d", &statusCode)
	if len(parts) >= 3 {
		statusText = parts[2]
	}

	// Parse headers
	headers := make(http.Header)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		headers.Add(key, value)
	}

	// Add marker header
	headers.Set("X-Mapped-Local", localPath)

	// Update Content-Length to match actual body
	headers.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	return &http.Response{
		StatusCode:    statusCode,
		Status:        fmt.Sprintf("%d %s", statusCode, statusText),
		Proto:         proto,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        headers,
		Body:          io.NopCloser(bytes.NewBuffer(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
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
