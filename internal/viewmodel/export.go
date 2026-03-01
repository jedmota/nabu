package viewmodel

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"nabu/internal/model"
)

// appVersion is the application version used in HAR exports.
// Set via SetAppVersion from main.
var appVersion = "dev"

// SetAppVersion sets the application version for HAR exports.
func SetAppVersion(v string) { appVersion = v }

// FormatCURL formats a flow as a cURL command string.
func FormatCURL(flow *model.Flow) (string, error) {
	if flow == nil || flow.Request == nil {
		return "", fmt.Errorf("no request to export")
	}
	if flow.Tunneled {
		return "", fmt.Errorf("cannot export tunneled connections")
	}

	req := flow.Request
	var parts []string

	parts = append(parts, "curl")

	if req.Method != "GET" {
		parts = append(parts, "-X", req.Method)
	}

	// Add headers (skip Host since curl sets it from URL)
	for key, values := range req.Headers {
		if strings.EqualFold(key, "Host") {
			continue
		}
		for _, v := range values {
			parts = append(parts, "-H", shellQuote(key+": "+v))
		}
	}

	// Add body
	if len(req.Body) > 0 {
		parts = append(parts, "-d", shellQuote(string(req.Body)))
	}

	parts = append(parts, shellQuote(req.URL))

	return strings.Join(parts, " "), nil
}

// HARLog is the top-level HAR structure.
type HARLog struct {
	Log HARLogInner `json:"log"`
}

// HARLogInner contains HAR metadata and entries.
type HARLogInner struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

// HARCreator identifies the tool.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry is a single request/response pair.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Timings         HARTimings  `json:"timings"`
}

// HARRequest represents an HTTP request in HAR format.
type HARRequest struct {
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	HTTPVersion string         `json:"httpVersion"`
	Headers     []HARNameValue `json:"headers"`
	QueryString []HARNameValue `json:"queryString"`
	BodySize    int            `json:"bodySize"`
	PostData    *HARPostData   `json:"postData,omitempty"`
}

// HARResponse represents an HTTP response in HAR format.
type HARResponse struct {
	Status      int            `json:"status"`
	StatusText  string         `json:"statusText"`
	HTTPVersion string         `json:"httpVersion"`
	Headers     []HARNameValue `json:"headers"`
	Content     HARContent     `json:"content"`
	BodySize    int            `json:"bodySize"`
}

// HARNameValue is a key-value pair.
type HARNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData holds request body data.
type HARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARContent holds response body data.
type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARTimings holds timing data.
type HARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// FormatHAR formats one or more flows as a HAR JSON string.
func FormatHAR(flows []*model.Flow) ([]byte, error) {
	har := HARLog{
		Log: HARLogInner{
			Version: "1.2",
			Creator: HARCreator{Name: "nabu", Version: appVersion},
			Entries: make([]HAREntry, 0, len(flows)),
		},
	}

	for _, flow := range flows {
		if flow == nil || flow.Request == nil || flow.Tunneled {
			continue
		}
		har.Log.Entries = append(har.Log.Entries, flowToHAREntry(flow))
	}

	return json.MarshalIndent(har, "", "  ")
}

func flowToHAREntry(flow *model.Flow) HAREntry {
	req := flow.Request
	entry := HAREntry{
		StartedDateTime: flow.StartTime.Format(time.RFC3339Nano),
		Time:            float64(flow.Duration().Milliseconds()),
		Request: HARRequest{
			Method:      req.Method,
			URL:         req.URL,
			HTTPVersion: req.Proto,
			Headers:     headersToHAR(req.Headers),
			QueryString: []HARNameValue{},
			BodySize:    len(req.Body),
		},
		Timings: HARTimings{
			Send:    0,
			Wait:    float64(flow.Duration().Milliseconds()),
			Receive: 0,
		},
	}

	if len(req.Body) > 0 {
		entry.Request.PostData = &HARPostData{
			MimeType: req.Headers.Get("Content-Type"),
			Text:     string(req.Body),
		}
	}

	if flow.Response != nil {
		resp := flow.Response
		statusText := resp.Status
		// Strip the status code prefix (e.g. "200 OK" -> "OK")
		if i := strings.IndexByte(statusText, ' '); i >= 0 {
			statusText = statusText[i+1:]
		}
		entry.Response = HARResponse{
			Status:      resp.StatusCode,
			StatusText:  statusText,
			HTTPVersion: resp.Proto,
			Headers:     headersToHAR(resp.Headers),
			Content: HARContent{
				Size:     len(resp.Body),
				MimeType: resp.Headers.Get("Content-Type"),
				Text:     string(resp.Body),
			},
			BodySize: len(resp.Body),
		}
	} else {
		entry.Response = HARResponse{
			Status:     0,
			StatusText: "Incomplete",
			Headers:    []HARNameValue{},
			Content:    HARContent{},
		}
	}

	return entry
}

func headersToHAR(headers map[string][]string) []HARNameValue {
	result := make([]HARNameValue, 0)
	for name, values := range headers {
		for _, v := range values {
			result = append(result, HARNameValue{Name: name, Value: v})
		}
	}
	return result
}

// ParseHAR parses HAR JSON data and returns flows.
func ParseHAR(data []byte) ([]*model.Flow, error) {
	var har HARLog
	if err := json.Unmarshal(data, &har); err != nil {
		return nil, fmt.Errorf("invalid HAR: %w", err)
	}

	flows := make([]*model.Flow, 0, len(har.Log.Entries))
	for _, entry := range har.Log.Entries {
		flow := harEntryToFlow(entry)
		if flow != nil {
			flows = append(flows, flow)
		}
	}
	return flows, nil
}

func harEntryToFlow(entry HAREntry) *model.Flow {
	startTime, _ := time.Parse(time.RFC3339Nano, entry.StartedDateTime)
	if startTime.IsZero() {
		startTime = time.Now()
	}
	endTime := startTime.Add(time.Duration(entry.Time) * time.Millisecond)

	// Parse URL for host and path
	host := ""
	path := ""
	if u, err := parseURL(entry.Request.URL); err == nil {
		host = u.Hostname()
		path = u.Path
		if u.RawQuery != "" {
			path += "?" + u.RawQuery
		}
	}

	flow := &model.Flow{
		StartTime: startTime,
		EndTime:   endTime,
		Request: &model.Request{
			Method:  entry.Request.Method,
			URL:     entry.Request.URL,
			Host:    host,
			Path:    path,
			Proto:   entry.Request.HTTPVersion,
			Headers: harToHeaders(entry.Request.Headers),
		},
	}

	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		flow.Request.Body = []byte(entry.Request.PostData.Text)
	}

	if entry.Response.Status > 0 {
		statusText := entry.Response.StatusText
		if statusText == "" {
			statusText = fmt.Sprintf("%d", entry.Response.Status)
		}
		flow.Response = &model.Response{
			StatusCode: entry.Response.Status,
			Status:     fmt.Sprintf("%d %s", entry.Response.Status, statusText),
			Proto:      entry.Response.HTTPVersion,
			Headers:    harToHeaders(entry.Response.Headers),
			Body:       []byte(entry.Response.Content.Text),
		}
	}

	return flow
}

func harToHeaders(nvs []HARNameValue) map[string][]string {
	headers := make(map[string][]string)
	for _, nv := range nvs {
		headers[nv.Name] = append(headers[nv.Name], nv.Value)
	}
	return headers
}

func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
