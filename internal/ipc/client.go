package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

const maxScanBuf = 10 * 1024 * 1024 // 10 MB

// Client connects to a primary instance's IPC server and populates
// a local FlowStore with the received data.
type Client struct {
	conn        net.Conn
	store       *proxy.FlowStore
	eventChan   chan model.FlowEvent
	mu          sync.RWMutex
	port        int
	bindAddress string
	done        chan struct{}
	disconnected chan struct{}
}

// NewClient connects to the Unix domain socket for the given port and
// runs the receive loop in the background.
func NewClient(port int) (*Client, error) {
	sockPath := SocketPath(port)
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("ipc: connect %s: %w", sockPath, err)
	}

	eventChan := make(chan model.FlowEvent, 1000)
	store := proxy.NewFlowStore(10000, nil) // nil primary channel; we use our own

	c := &Client{
		conn:         conn,
		store:        store,
		eventChan:    eventChan,
		port:         port,
		done:         make(chan struct{}),
		disconnected: make(chan struct{}),
	}

	go c.readLoop()
	return c, nil
}

// Events returns the event channel (satisfies FlowSource via Adapter).
func (c *Client) Events() <-chan model.FlowEvent {
	return c.eventChan
}

// FlowStore returns the local flow store.
func (c *Client) FlowStore() *proxy.FlowStore {
	return c.store
}

// Port returns the proxy port received from the primary.
func (c *Client) Port() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.port
}

// BindAddress returns the bind address received from the primary.
func (c *Client) BindAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bindAddress
}

// Disconnected returns a channel that is closed when the primary disconnects.
func (c *Client) Disconnected() <-chan struct{} {
	return c.disconnected
}

// Close shuts down the client connection.
func (c *Client) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.conn.Close()
}

func (c *Client) readLoop() {
	defer func() {
		select {
		case <-c.disconnected:
		default:
			close(c.disconnected)
		}
	}()

	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, maxScanBuf), maxScanBuf)

	for scanner.Scan() {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := UnmarshalMessage(scanner.Bytes())
		if err != nil {
			continue
		}

		switch msg.Type {
		case "hello":
			var p HelloPayload
			if err := json.Unmarshal(msg.Payload, &p); err == nil {
				c.mu.Lock()
				c.port = p.Port
				c.bindAddress = p.BindAddress
				c.mu.Unlock()
			}

		case "sync":
			var p SyncPayload
			if err := json.Unmarshal(msg.Payload, &p); err == nil {
				for _, fw := range p.Flows {
					flow := WireToFlow(fw)
					c.store.AddWithID(flow)
					c.emitEvent(model.FlowEvent{Type: model.FlowEventRequest, Flow: flow})
					if flow.Response != nil {
						c.emitEvent(model.FlowEvent{Type: model.FlowEventResponse, Flow: flow})
					}
					if flow.Error != nil {
						c.emitEvent(model.FlowEvent{Type: model.FlowEventError, Flow: flow})
					}
				}
			}

		case "sync_done":
			// Nothing special to do; we already populated the store.

		case "flow_event":
			var p FlowEventPayload
			if err := json.Unmarshal(msg.Payload, &p); err == nil {
				flow := WireToFlow(p.Flow)
				switch p.EventType {
				case model.FlowEventRequest:
					c.store.AddWithID(flow)
				default:
					c.store.UpdateFromRemote(flow, p.EventType)
				}
				c.emitEvent(model.FlowEvent{Type: p.EventType, Flow: flow})
			}
		}
	}
}

func (c *Client) emitEvent(evt model.FlowEvent) {
	select {
	case c.eventChan <- evt:
	default:
	}
}

// SendConfigReload tells the primary instance to reload its configs from disk.
func (c *Client) SendConfigReload() error {
	data, err := MarshalMessage("config_reload", ConfigReloadPayload{})
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.conn.Write(data)
	return err
}
