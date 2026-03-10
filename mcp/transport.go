package mcp

import (
	"io"
	"net/http"
	"os"
	"os/exec"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type StdioConfig struct {
	Command string
	Args    []string
	Env     []string
	Stderr  io.Writer
	Cwd     string
}

func NewStdioClientTransport(cfg StdioConfig) sdkmcp.Transport {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	if cfg.Cwd != "" {
		cmd.Dir = cfg.Cwd
	}
	if len(cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), cfg.Env...)
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	}

	return &sdkmcp.CommandTransport{Command: cmd}
}

type SSETransportConfig struct {
	URL        string
	HTTPClient *http.Client
}

func NewSSEClientTransport(cfg SSETransportConfig) sdkmcp.Transport {
	return &sdkmcp.SSEClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: cfg.HTTPClient,
	}
}

type HTTPTransportConfig struct {
	URL        string
	HTTPClient *http.Client
	MaxRetries int
}

func NewHTTPClientTransport(cfg HTTPTransportConfig) sdkmcp.Transport {
	return &sdkmcp.StreamableClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: cfg.HTTPClient,
		MaxRetries: cfg.MaxRetries,
	}
}
