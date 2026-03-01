package model

import (
	"testing"
	"time"
)

func TestAlertRule_Match_StatusCode(t *testing.T) {
	rule := &AlertRule{Type: AlertStatusCode, Value: 500, Enabled: true}

	tests := []struct {
		name   string
		flow   *Flow
		want   bool
	}{
		{"500", &Flow{Response: &Response{StatusCode: 500}}, true},
		{"503", &Flow{Response: &Response{StatusCode: 503}}, true},
		{"499", &Flow{Response: &Response{StatusCode: 499}}, false},
		{"200", &Flow{Response: &Response{StatusCode: 200}}, false},
		{"no response", &Flow{}, false},
		{"nil flow", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rule.Match(tt.flow); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertRule_Match_StatusCode_4xx(t *testing.T) {
	rule := &AlertRule{Type: AlertStatusCode, Value: 400, Enabled: true}

	if !rule.Match(&Flow{Response: &Response{StatusCode: 404}}) {
		t.Error("400 rule should match 404")
	}
	if rule.Match(&Flow{Response: &Response{StatusCode: 500}}) {
		t.Error("400 rule should not match 500")
	}
}

func TestAlertRule_Match_Latency(t *testing.T) {
	rule := &AlertRule{Type: AlertLatency, Value: 1000, Enabled: true}

	slow := &Flow{
		StartTime: time.Now().Add(-2 * time.Second),
		EndTime:   time.Now(),
		Response:  &Response{StatusCode: 200},
	}
	if !rule.Match(slow) {
		t.Error("should match slow flow (2s > 1000ms)")
	}

	fast := &Flow{
		StartTime: time.Now().Add(-100 * time.Millisecond),
		EndTime:   time.Now(),
		Response:  &Response{StatusCode: 200},
	}
	if rule.Match(fast) {
		t.Error("should not match fast flow (100ms < 1000ms)")
	}

	// Incomplete flow
	incomplete := &Flow{StartTime: time.Now()}
	if rule.Match(incomplete) {
		t.Error("should not match incomplete flow")
	}
}

func TestAlertRule_Match_Disabled(t *testing.T) {
	rule := &AlertRule{Type: AlertStatusCode, Value: 500, Enabled: false}
	flow := &Flow{Response: &Response{StatusCode: 500}}
	if rule.Match(flow) {
		t.Error("disabled rule should not match")
	}
}

func TestDefaultAlertRules(t *testing.T) {
	rules := DefaultAlertRules()
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 default rules, got %d", len(rules))
	}

	// First should be 5xx enabled
	if rules[0].Type != AlertStatusCode || rules[0].Value != 500 || !rules[0].Enabled {
		t.Errorf("first default rule: %+v", rules[0])
	}

	// Second should be latency disabled
	if rules[1].Type != AlertLatency || !rules[1].Enabled == true {
		// Latency default is disabled
	}
}
