package debug

import (
	"encoding/json"
	"net"
	"log"
)

// IPCClient sends debug interactions via Unix socket to standalone debug server
type IPCClient struct {
	socketPath string
}

// NewIPCClient creates a new IPC client for sending debug data
func NewIPCClient(socketPath string) *IPCClient {
	return &IPCClient{socketPath: socketPath}
}

// SendInteraction sends an interaction to the debug server via IPC
func (c *IPCClient) SendInteraction(interaction DebugInteraction) error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		// Don't fail if debug server isn't running
		log.Printf("Debug IPC connection failed: %v", err)
		return nil
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(interaction); err != nil {
		log.Printf("Debug IPC encode failed: %v", err)
		return nil
	}
	
	log.Printf("Debug interaction sent: %s", interaction.ToolName)
	return nil
}

// IPCServer receives debug interactions via Unix socket
type IPCServer struct {
	socketPath  string
	debugServer *DebugServer
	listener    net.Listener
}

// NewIPCServer creates a new IPC server
func NewIPCServer(socketPath string, debugServer *DebugServer) *IPCServer {
	return &IPCServer{
		socketPath:  socketPath,
		debugServer: debugServer,
	}
}

// Start starts the IPC server
func (s *IPCServer) Start() error {
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single IPC connection
func (s *IPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	var interaction DebugInteraction
	
	if err := decoder.Decode(&interaction); err != nil {
		log.Printf("IPC decode error: %v", err)
		return
	}
	
	s.debugServer.AddInteraction(interaction)
}

// Stop stops the IPC server
func (s *IPCServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}