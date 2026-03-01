package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"nabu/internal/model"
	"nabu/internal/util"
)

// MapLocalResponse represents the JSONC file structure for mapped responses
type MapLocalResponse struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
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

// parseJSONCResponseFile parses a JSONC response file
func parseJSONCResponseFile(content []byte, req *http.Request, localPath string) *http.Response {
	jsonContent := util.StripJSONComments(content)

	var resp MapLocalResponse
	if err := json.Unmarshal(jsonContent, &resp); err != nil {
		return nil
	}

	if resp.Status == 0 {
		resp.Status = 200
	}

	return buildResponseFromJSON(&resp, req, localPath)
}

// buildResponseFromJSON creates an HTTP response from parsed JSONC
func buildResponseFromJSON(resp *MapLocalResponse, req *http.Request, localPath string) *http.Response {
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
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		case []interface{}:
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		default:
			var err error
			body, err = json.Marshal(v)
			if err != nil {
				body = []byte{}
			}
		}
	}

	headers := make(http.Header)
	if resp.Headers != nil {
		for key, value := range resp.Headers {
			headers.Set(key, value)
		}
	}

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
	headerEnd := bytes.Index(content, []byte("\n\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(content, []byte("\r\n\r\n"))
		if headerEnd == -1 {
			return nil
		}
		headerEnd += 2
	}

	headerSection := string(content[:headerEnd])
	body := content[headerEnd+2:]

	lines := strings.Split(headerSection, "\n")
	if len(lines) == 0 {
		return nil
	}

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

	headers.Set("X-Mapped-Local", localPath)
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
