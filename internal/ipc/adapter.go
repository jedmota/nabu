package ipc

import (
	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

// Adapter wraps an IPC Client to satisfy the proxy.FlowSource interface,
// allowing a secondary instance to be used exactly like a primary.
type Adapter struct {
	client       *Client
	sslProxyList *proxy.SSLProxyList
	mapRules     *model.MapRuleStore
}

// NewAdapter creates a FlowSource adapter around an IPC client.
func NewAdapter(client *Client) *Adapter {
	return &Adapter{
		client:       client,
		sslProxyList: proxy.NewSSLProxyList(),
		mapRules:     model.NewMapRuleStore(),
	}
}

func (a *Adapter) Events() <-chan model.FlowEvent {
	return a.client.Events()
}

func (a *Adapter) FlowStore() *proxy.FlowStore {
	return a.client.FlowStore()
}

func (a *Adapter) SSLProxyList() *proxy.SSLProxyList {
	return a.sslProxyList
}

func (a *Adapter) MapRules() *model.MapRuleStore {
	return a.mapRules
}

func (a *Adapter) Port() int {
	return a.client.Port()
}

func (a *Adapter) BindAddress() string {
	return a.client.BindAddress()
}

// Disconnected exposes the client's disconnection channel.
func (a *Adapter) Disconnected() <-chan struct{} {
	return a.client.Disconnected()
}

// NotifyConfigChange tells the primary instance to reload configs from disk.
func (a *Adapter) NotifyConfigChange() {
	a.client.SendConfigReload()
}

// Close shuts down the underlying client.
func (a *Adapter) Close() {
	a.client.Close()
}
