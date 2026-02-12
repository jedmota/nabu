package viewmodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"proxy-tui/internal/model"
	"proxy-tui/internal/util"
)

// FormatFlowSummary returns a formatted summary of a flow for the list
func (vm *ViewModel) FormatFlowSummary(flow *model.Flow) (method, host, path, status, duration string) {
	if flow == nil || flow.Request == nil {
		return "", "", "", "", ""
	}

	method = flow.Request.Method
	host = flow.Request.Host
	path = flow.Request.Path
	if path == "" {
		path = "/"
	}

	if flow.Response != nil {
		status = fmt.Sprintf("%d", flow.Response.StatusCode)
	} else if flow.Error != nil {
		status = "ERR"
	} else {
		status = "..."
	}

	if flow.IsComplete() {
		duration = formatDuration(flow.Duration())
	} else {
		duration = "..."
	}

	return
}

// FormatFlowDetail returns a detailed formatted view of a flow
func (vm *ViewModel) FormatFlowDetail(flow *model.Flow, raw bool) string {
	if flow == nil || flow.Request == nil {
		return "No flow selected"
	}

	// Handle tunneled flows
	if flow.Tunneled {
		var sb strings.Builder
		sb.WriteString("[yellow]═══ Tunneled Connection ═══[-]\n\n")
		sb.WriteString(fmt.Sprintf("[gray]CONNECT[-] %s\n\n", flow.Request.Host))
		sb.WriteString("[gray]This connection was tunneled without SSL interception.[-]\n")
		sb.WriteString("[gray]The request and response content is encrypted.[-]\n\n")
		sb.WriteString("To inspect this traffic, add the host to the whitelist:\n")
		sb.WriteString(fmt.Sprintf("  Press [yellow]w[-] and enter: [green]%s[-]\n", flow.Request.Host))
		sb.WriteString(fmt.Sprintf("  Or use wildcard: [green]*.%s[-]\n", getBaseDomain(flow.Request.Host)))
		return sb.String()
	}

	var sb strings.Builder

	// Request section
	sb.WriteString("[yellow]═══ Request ═══[-]\n\n")
	sb.WriteString(fmt.Sprintf("[gray]%s[-]\n", flow.StartTime.Format("2006-01-02 15:04:05.000")))
	sb.WriteString(fmt.Sprintf("[green]%s[-] %s %s\n", flow.Request.Method, flow.Request.URL, flow.Request.Proto))
	sb.WriteString(fmt.Sprintf("Host: %s\n", flow.Request.Host))

	// Request headers
	sb.WriteString("\n[blue]Headers:[-]\n")
	for key, values := range flow.Request.Headers {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// Request body
	if len(flow.Request.Body) > 0 {
		sb.WriteString("\n[blue]Body:[-]\n")
		if raw {
			sb.WriteString(string(flow.Request.Body))
		} else {
			sb.WriteString(formatBody(flow.Request.Body, flow.Request.Headers.Get("Content-Type")))
		}
		sb.WriteString("\n")
	}

	// Response section
	if flow.Response != nil {
		sb.WriteString("\n[yellow]═══ Response ═══[-]\n\n")
		sb.WriteString(fmt.Sprintf("[cyan]%s[-] %s\n", flow.Response.Status, flow.Response.Proto))

		// Response headers
		sb.WriteString("\n[blue]Headers:[-]\n")
		for key, values := range flow.Response.Headers {
			for _, value := range values {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}

		// Response body
		if len(flow.Response.Body) > 0 {
			sb.WriteString("\n[blue]Body:[-]\n")
			if raw {
				sb.WriteString(string(flow.Response.Body))
			} else {
				sb.WriteString(formatBody(flow.Response.Body, flow.Response.Headers.Get("Content-Type")))
			}
			sb.WriteString("\n")
		}
	} else if flow.Error != nil {
		sb.WriteString("\n[red]═══ Error ═══[-]\n\n")
		sb.WriteString(flow.Error.Error())
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n[gray]Waiting for response...[-]\n")
	}

	// Timing info
	sb.WriteString("\n[yellow]═══ Timing ═══[-]\n\n")
	sb.WriteString(fmt.Sprintf("Started: %s\n", flow.StartTime.Format("15:04:05.000")))
	if flow.IsComplete() {
		sb.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(flow.Duration())))
	}

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d interface{ Milliseconds() int64 }) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000)
}

// formatBody formats the body based on content type
func formatBody(body []byte, contentType string) string {
	maxLen := 10000

	// Try to pretty-print JSON
	if strings.Contains(contentType, "json") || util.IsJSON(body) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			result := prettyJSON.String()
			if len(result) > maxLen {
				return result[:maxLen] + "\n... (truncated)"
			}
			return result
		}
	}

	// Default: return as string
	if len(body) > maxLen {
		return string(body[:maxLen]) + "\n... (truncated)"
	}
	return string(body)
}

// getBaseDomain extracts the base domain from a host (e.g., "api.example.com" -> "example.com")
func getBaseDomain(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	// Return last two parts
	return strings.Join(parts[len(parts)-2:], ".")
}
