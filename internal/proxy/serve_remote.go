package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nabu/internal/model"
)

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
