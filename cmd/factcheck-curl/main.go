package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

// MCP JSON-RPC message types
type Request struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Tool call parameters
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func main() {
	var (
		serverCmd = flag.String("cmd", "./bin/mcp-factcheck-server", "Command to run MCP server")
		dataDir   = flag.String("data-dir", "./embeddings", "Data directory for server")
		timeout   = flag.Duration("timeout", 30*time.Second, "Request timeout")
	)
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> [args...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  initialize                    - Initialize MCP connection\n")
		fmt.Fprintf(os.Stderr, "  tools/list                    - List available tools\n")
		fmt.Fprintf(os.Stderr, "  tools/call <tool> <args>      - Call a tool with JSON arguments\n")
		fmt.Fprintf(os.Stderr, "  resources/list                - List available resources\n")
		fmt.Fprintf(os.Stderr, "  resources/read <uri>          - Read a resource\n")
		fmt.Fprintf(os.Stderr, "  prompts/list                  - List available prompts\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s tools/list\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s tools/call validate_content '{\"content\":\"MCP uses JSON-RPC\"}'\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s tools/call search_spec '{\"query\":\"tools\",\"top_k\":3}'\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s tools/call list_spec_versions '{}'\n", os.Args[0])
		os.Exit(1)
	}

	command := flag.Args()[0]
	args := flag.Args()[1:]

	client, err := NewMCPClient(*serverCmd, *dataDir, *timeout)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	defer client.Close()

	switch command {
	case "initialize":
		err = client.Initialize()
	case "tools/list":
		err = client.ListTools()
	case "tools/call":
		if len(args) < 2 {
			log.Fatalf("tools/call requires tool name and arguments")
		}
		err = client.CallTool(args[0], args[1])
	case "resources/list":
		err = client.ListResources()
	case "resources/read":
		if len(args) < 1 {
			log.Fatalf("resources/read requires URI")
		}
		err = client.ReadResource(args[0])
	case "prompts/list":
		err = client.ListPrompts()
	default:
		log.Fatalf("Unknown command: %s", command)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

type MCPClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	timeout time.Duration
	id      int
}

func NewMCPClient(serverCmd, dataDir string, timeout time.Duration) (*MCPClient, error) {
	cmd := exec.Command(serverCmd, "--data-dir", dataDir)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	client := &MCPClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewScanner(stdout),
		timeout: timeout,
		id:      1,
	}

	// Initialize the connection
	if err := client.Initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return client, nil
}

func (c *MCPClient) Close() {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
}

func (c *MCPClient) sendRequest(method string, params any) (*Response, error) {
	req := Request{
		Jsonrpc: "2.0",
		ID:      c.id,
		Method:  method,
		Params:  params,
	}
	c.id++

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := c.stdin.Write(append(reqData, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response with timeout
	responseChan := make(chan string, 1)
	go func() {
		if c.stdout.Scan() {
			responseChan <- c.stdout.Text()
		} else {
			responseChan <- ""
		}
	}()

	select {
	case responseText := <-responseChan:
		if responseText == "" {
			return nil, fmt.Errorf("no response received")
		}

		var resp Response
		if err := json.Unmarshal([]byte(responseText), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return &resp, nil
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("request timeout")
	}
}

func (c *MCPClient) Initialize() error {
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"roots": map[string]any{
				"listChanged": false,
			},
		},
		"clientInfo": map[string]any{
			"name":    "factcheck-curl",
			"version": "0.1.0",
		},
	}

	resp, err := c.sendRequest("initialize", initParams)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	initReq := Request{
		Jsonrpc: "2.0",
		Method:  "initialized",
	}
	initData, _ := json.Marshal(initReq)
	c.stdin.Write(append(initData, '\n'))

	return nil
}

func (c *MCPClient) ListTools() error {
	resp, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	output, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func (c *MCPClient) CallTool(toolName, argsJSON string) error {
	var toolArgs map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &toolArgs); err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	callParams := CallToolParams{
		Name:      toolName,
		Arguments: toolArgs,
	}

	resp, err := c.sendRequest("tools/call", callParams)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}

	output, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func (c *MCPClient) ListResources() error {
	resp, err := c.sendRequest("resources/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("resources/list error: %s", resp.Error.Message)
	}

	output, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func (c *MCPClient) ReadResource(uri string) error {
	resourceParams := map[string]any{
		"uri": uri,
	}

	resp, err := c.sendRequest("resources/read", resourceParams)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("resources/read error: %s", resp.Error.Message)
	}

	output, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func (c *MCPClient) ListPrompts() error {
	resp, err := c.sendRequest("prompts/list", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("prompts/list error: %s", resp.Error.Message)
	}

	output, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(output))
	return nil
}