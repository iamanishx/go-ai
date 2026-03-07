package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/iamanishx/go-ai/provider"
)

type StreamReader struct {
	parts  chan provider.StreamPart
	err    error
	closed bool
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewStreamReader(ctx context.Context) *StreamReader {
	ctx, cancel := context.WithCancel(ctx)
	return &StreamReader{
		parts:  make(chan provider.StreamPart, 100),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *StreamReader) Part() <-chan provider.StreamPart {
	return s.parts
}

func (s *StreamReader) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.parts)
		s.cancel()
	}
	return nil
}

func (s *StreamReader) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

func (s *StreamReader) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *StreamReader) WritePart(part provider.StreamPart) {
	select {
	case s.parts <- part:
	case <-s.ctx.Done():
	}
}

type TextStreamWriter struct {
	mu       sync.RWMutex
	chunks   []string
	closed   bool
	onChunk  func(string)
	onFinish func(string)
}

func NewTextStreamWriter() *TextStreamWriter {
	return &TextStreamWriter{
		chunks: make([]string, 0),
	}
}

func (w *TextStreamWriter) WriteChunk(chunk string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.chunks = append(w.chunks, chunk)
	if w.onChunk != nil {
		w.onChunk(chunk)
	}
}

func (w *TextStreamWriter) Text() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return strings.Join(w.chunks, "")
}

func (w *TextStreamWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	if w.onFinish != nil {
		w.onFinish(w.Text())
	}
}

func (w *TextStreamWriter) OnChunk(f func(string)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onChunk = f
}

func (w *TextStreamWriter) OnFinish(f func(string)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onFinish = f
}

func ParseSSEStream(body io.Reader) ([]provider.StreamPart, error) {
	decoder := json.NewDecoder(body)
	var parts []provider.StreamPart

	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		part := provider.StreamPart{}
		if v, ok := raw["type"].(string); ok {
			part.Type = v
		}
		if v, ok := raw["text"].(string); ok {
			part.Text = v
		}
		if v, ok := raw["textDelta"].(string); ok {
			part.Text = v
		}
		if v, ok := raw["toolCallId"].(string); ok {
			part.ToolCallID = v
		}
		if v, ok := raw["toolName"].(string); ok {
			part.ToolName = v
		}
		if v, ok := raw["input"].(string); ok {
			part.ToolInput = v
		}
		if v, ok := raw["inputDelta"].(string); ok {
			part.ToolInputDelta = v
		}
		if v, ok := raw["output"].(string); ok {
			part.ToolResult = v
		}
		if v, ok := raw["finishReason"].(string); ok {
			part.FinishReason = v
		}
		if usage, ok := raw["usage"].(map[string]interface{}); ok {
			if t, ok := usage["completionTokens"].(float64); ok {
				part.Usage.OutputTokens = int(t)
			}
			if t, ok := usage["promptTokens"].(float64); ok {
				part.Usage.InputTokens = int(t)
			}
			if t, ok := usage["totalTokens"].(float64); ok {
				part.Usage.TotalTokens = int(t)
			}
		}

		parts = append(parts, part)
	}

	return parts, nil
}

type StreamOptions struct {
	Headers     http.Header
	Timeout     time.Duration
	AbortSignal <-chan struct{}
}

func StreamFromResponse(ctx context.Context, url string, body io.Reader, opts StreamOptions) (*StreamReader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = opts.Headers
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	if opts.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
		req = req.WithContext(ctx)
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stream request failed with status %d: %s", resp.StatusCode, string(body))
	}

	reader := NewStreamReader(req.Context())

	go func() {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)

		for {
			var raw map[string]interface{}
			if err := decoder.Decode(&raw); err != nil {
				if err == io.EOF {
					break
				}
				reader.SetError(err)
				reader.Close()
				return
			}

			part := parseSSEPart(raw)
			reader.WritePart(part)
		}

		reader.Close()
	}()

	return reader, nil
}

func parseSSEPart(raw map[string]interface{}) provider.StreamPart {
	part := provider.StreamPart{}

	if v, ok := raw["type"].(string); ok {
		part.Type = v
	}

	switch part.Type {
	case "text-delta", "text-start", "text-end":
		if v, ok := raw["text"].(string); ok {
			part.Text = v
		}
		if v, ok := raw["textDelta"].(string); ok {
			part.Text = v
		}

	case "tool-call", "tool-input-start":
		if v, ok := raw["toolCallId"].(string); ok {
			part.ToolCallID = v
		}
		if v, ok := raw["toolName"].(string); ok {
			part.ToolName = v
		}

	case "tool-input-delta":
		if v, ok := raw["toolCallId"].(string); ok {
			part.ToolCallID = v
		}
		if v, ok := raw["inputDelta"].(string); ok {
			part.ToolInputDelta = v
		}

	case "tool-result":
		if v, ok := raw["toolCallId"].(string); ok {
			part.ToolCallID = v
		}
		if v, ok := raw["result"].(string); ok {
			part.ToolResult = v
		}

	case "finish":
		if v, ok := raw["finishReason"].(string); ok {
			part.FinishReason = v
		}
		if usage, ok := raw["usage"].(map[string]interface{}); ok {
			if t, ok := usage["completionTokens"].(float64); ok {
				part.Usage.OutputTokens = int(t)
			}
			if t, ok := usage["promptTokens"].(float64); ok {
				part.Usage.InputTokens = int(t)
			}
			if t, ok := usage["totalTokens"].(float64); ok {
				part.Usage.TotalTokens = int(t)
			}
		}

	case "error":
		if v, ok := raw["error"].(string); ok {
			part.Error = fmt.Errorf(v)
		}
	}

	return part
}

func ConsumeTextStream(parts <-chan provider.StreamPart) string {
	var text strings.Builder
	for part := range parts {
		if part.Type == "text-delta" {
			text.WriteString(part.Text)
		}
	}
	return text.String()
}
