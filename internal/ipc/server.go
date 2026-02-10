package ipc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"

	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

const syncBatchSize = 100

// Server is the IPC server that the primary instance runs.
// It listens on a Unix domain socket and streams flow data to
// connected secondary instances.
type Server struct {
	listener      net.Listener
	source        proxy.FlowSource
	sockPath      string
	clients       map[net.Conn]struct{}
	clientsMu     sync.Mutex
	done          chan struct{}
	configReload  chan struct{}
}

// NewServer creates and starts a new IPC server.
func NewServer(source proxy.FlowSource, port int) (*Server, error) {
	sockPath := SocketPath(port)

	// Clean up any stale socket
	os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen %s: %w", sockPath, err)
	}

	s := &Server{
		listener:     ln,
		source:       source,
		sockPath:     sockPath,
		clients:      make(map[net.Conn]struct{}),
		done:         make(chan struct{}),
		configReload: make(chan struct{}, 10),
	}

	go s.acceptLoop()
	return s, nil
}

// Stop shuts down the IPC server and removes the socket file.
func (s *Server) Stop() {
	close(s.done)
	s.listener.Close()
	os.Remove(s.sockPath)

	// Close all client connections
	s.clientsMu.Lock()
	for c := range s.clients {
		c.Close()
	}
	s.clientsMu.Unlock()
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			continue
		}
		s.clientsMu.Lock()
		s.clients[conn] = struct{}{}
		s.clientsMu.Unlock()

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	store := s.source.FlowStore()

	// 1. Send hello
	hello := HelloPayload{
		Port:        s.source.Port(),
		BindAddress: s.source.BindAddress(),
		FlowCount:   store.Count(),
	}
	if err := s.writeMessage(conn, "hello", hello); err != nil {
		return
	}

	// 2. Subscribe to events BEFORE we start the sync so we don't miss any.
	sub := store.Subscribe()
	defer store.Unsubscribe(sub)

	// 3. Send existing flows in batches
	allFlows := store.All()
	totalBatches := (len(allFlows) + syncBatchSize - 1) / syncBatchSize
	if totalBatches == 0 {
		totalBatches = 1
	}
	for batch := 0; batch*syncBatchSize < len(allFlows); batch++ {
		start := batch * syncBatchSize
		end := start + syncBatchSize
		if end > len(allFlows) {
			end = len(allFlows)
		}
		wires := make([]FlowWire, 0, end-start)
		for _, f := range allFlows[start:end] {
			wires = append(wires, FlowToWire(f))
		}
		payload := SyncPayload{
			Flows:      wires,
			Batch:      batch + 1,
			TotalBatch: totalBatches,
		}
		if err := s.writeMessage(conn, "sync", payload); err != nil {
			return
		}
	}

	// 4. Send sync_done
	if err := s.writeMessage(conn, "sync_done", SyncDonePayload{}); err != nil {
		return
	}

	// 5. Start reading commands from the client
	clientDone := make(chan struct{})
	go s.readClient(conn, clientDone)

	// 6. Stream real-time events
	for {
		select {
		case <-s.done:
			return
		case <-clientDone:
			return
		case evt, ok := <-sub:
			if !ok {
				return
			}
			payload := FlowEventPayload{
				EventType: evt.Type,
				Flow:      FlowToWire(evt.Flow),
			}
			if err := s.writeMessage(conn, "flow_event", payload); err != nil {
				return
			}
		}
	}
}

func (s *Server) readClient(conn net.Conn, done chan struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	for scanner.Scan() {
		msg, err := UnmarshalMessage(scanner.Bytes())
		if err != nil {
			continue
		}
		switch msg.Type {
		case "config_reload":
			select {
			case s.configReload <- struct{}{}:
			default:
			}
		}
	}
}

func (s *Server) writeMessage(conn net.Conn, msgType string, payload interface{}) error {
	data, err := MarshalMessage(msgType, payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

// ConfigReloads returns a channel that receives a signal when a secondary
// instance requests that the primary reload its configuration from disk.
func (s *Server) ConfigReloads() <-chan struct{} {
	return s.configReload
}

// FlowEvent re-exports for convenience in type assertions
type FlowEvent = model.FlowEvent
