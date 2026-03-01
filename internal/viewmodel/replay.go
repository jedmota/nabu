package viewmodel

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"nabu/internal/model"
)

// ReplayFlow re-sends the given flow's request through the proxy,
// which captures it as a new flow automatically.
func (vm *ViewModel) ReplayFlow(flow *model.Flow) error {
	if flow == nil || flow.Request == nil {
		return fmt.Errorf("no request to replay")
	}
	if flow.Tunneled {
		return fmt.Errorf("cannot replay tunneled connections")
	}

	req := flow.Request

	proxyURL := fmt.Sprintf("http://%s:%d", vm.proxyHost(), vm.source.Port())

	transport := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy original headers
	for key, values := range req.Headers {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}
	httpReq.Host = req.Host

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("replay failed: %w", err)
	}
	resp.Body.Close()

	return nil
}

// proxyHost returns the host to connect to for replay.
// If the proxy binds to 0.0.0.0, use localhost instead.
func (vm *ViewModel) proxyHost() string {
	addr := vm.source.BindAddress()
	if addr == "0.0.0.0" || addr == "" || strings.HasPrefix(addr, "::") {
		return "127.0.0.1"
	}
	return addr
}
