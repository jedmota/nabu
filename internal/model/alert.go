package model

import "time"

// AlertType identifies the kind of alert condition.
type AlertType string

const (
	AlertStatusCode AlertType = "status_code"
	AlertLatency    AlertType = "latency"
)

// AlertRule defines a condition that triggers an alert on matching flows.
type AlertRule struct {
	Type    AlertType `json:"type"`
	Enabled bool      `json:"enabled"`
	// For AlertStatusCode: minimum status code (e.g. 500 matches 5xx)
	// For AlertLatency: threshold in milliseconds
	Value int `json:"value"`
}

// Match checks whether a flow triggers this alert rule.
func (r *AlertRule) Match(flow *Flow) bool {
	if !r.Enabled || flow == nil {
		return false
	}

	switch r.Type {
	case AlertStatusCode:
		if flow.Response == nil {
			return false
		}
		// Match status codes >= the configured value
		// e.g. value=500 matches 500-599, value=400 matches 400-499
		base := (r.Value / 100) * 100
		return flow.Response.StatusCode >= base && flow.Response.StatusCode < base+100

	case AlertLatency:
		if !flow.IsComplete() {
			return false
		}
		return flow.Duration() > time.Duration(r.Value)*time.Millisecond
	}

	return false
}

// DefaultAlertRules returns the built-in alert rules.
func DefaultAlertRules() []AlertRule {
	return []AlertRule{
		{Type: AlertStatusCode, Value: 500, Enabled: true},
		{Type: AlertLatency, Value: 5000, Enabled: false},
	}
}
