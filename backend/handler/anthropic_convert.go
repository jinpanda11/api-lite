package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"new-api-lite/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Anthropic → OpenAI request conversion ──────────────────────────────────

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []anthropicContentBlock
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicRequest struct {
	Model      string             `json:"model"`
	MaxTokens  int                `json:"max_tokens"`
	Messages   []anthropicMessage `json:"messages"`
	System     any                `json:"system"` // string or []anthropicContentBlock
	Stream     bool               `json:"stream"`
	Stop       any                `json:"stop_reason,omitempty"`
}

type openAIRequest struct {
	Model     string         `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int            `json:"max_tokens,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
	Stop      any            `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func anthropicToOpenAI(body []byte) ([]byte, error) {
	var areq anthropicRequest
	if err := json.Unmarshal(body, &areq); err != nil {
		return nil, err
	}

	omsgs := make([]openAIMessage, 0, len(areq.Messages)+1)

	// Anthropic top-level system → OpenAI system message
	if areq.System != nil {
		sysContent := extractTextContent(areq.System)
		if sysContent != "" {
			omsgs = append(omsgs, openAIMessage{Role: "system", Content: sysContent})
		}
	}

	for _, m := range areq.Messages {
		content := extractTextContent(m.Content)
		if content == "" {
			content = "..." // Anthropic requires non-empty, be safe
		}
		// Map Anthropic → OpenAI roles: "assistant" stays, "user" stays
		omsgs = append(omsgs, openAIMessage{Role: m.Role, Content: content})
	}

	oreq := openAIRequest{
		Model:     areq.Model,
		Messages:  omsgs,
		MaxTokens: areq.MaxTokens,
		Stream:    areq.Stream,
		Stop:      areq.Stop,
	}

	return json.Marshal(oreq)
}

func extractTextContent(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		var parts []string
		for _, block := range val {
			if b, ok := block.(map[string]any); ok {
				if t, ok := b["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// ── OpenAI → Anthropic response conversion ─────────────────────────────────

type anthropicResponse struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Role       string                 `json:"role"`
	Model      string                 `json:"model"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      anthropicUsage         `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func openAIResponseToAnthropic(body []byte) ([]byte, error) {
	var oresp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &oresp); err != nil {
		return nil, err
	}

	stopReason := "end_turn"
	if len(oresp.Choices) > 0 && oresp.Choices[0].FinishReason != "" {
		stopReason = mapStopReason(oresp.Choices[0].FinishReason)
	}

	content := []anthropicContentBlock{}
	if len(oresp.Choices) > 0 && oresp.Choices[0].Message.Content != "" {
		content = append(content, anthropicContentBlock{Type: "text", Text: oresp.Choices[0].Message.Content})
	}

	aresp := anthropicResponse{
		ID:         oresp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      oresp.Model,
		Content:    content,
		StopReason: stopReason,
		Usage: anthropicUsage{
			InputTokens:  oresp.Usage.PromptTokens,
			OutputTokens: oresp.Usage.CompletionTokens,
		},
	}

	return json.Marshal(aresp)
}

// openAIStreamChunk represents a single SSE data chunk from OpenAI streaming API.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}


func mapStopReason(openaiReason string) string {
	switch openaiReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// ── Non-streaming handler for Anthropic /v1/messages ────────────────────────

func handleAnthropicNonStream(c *gin.Context, resp *http.Response, user *model.User,
	token *model.Token, channel *model.Channel, modelName string, startTime time.Time) {

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		c.JSON(http.StatusBadGateway, openAIError("failed to read upstream response"))
		return
	}

	// Convert OpenAI response → Anthropic format
	anthropicBody, err := openAIResponseToAnthropic(bodyBytes)
	if err != nil {
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}

	// Extract token counts from OpenAI response for billing
	var promptTokens, completionTokens int
	var cr completionResponse
	if json.Unmarshal(bodyBytes, &cr) == nil {
		promptTokens = cr.Usage.PromptTokens
		completionTokens = cr.Usage.CompletionTokens
	}

	go recordLog(user, token, channel, modelName, promptTokens, completionTokens, 0,
		resp.StatusCode, c.Param("path"), startTime)

	c.Data(resp.StatusCode, "application/json", anthropicBody)
}

// ── Anthropic SSE event types ─────────────────────────────────────────────

type anthropicSSEMsgStart struct {
	Type    string                     `json:"type"`
	Message anthropicSSEMessage        `json:"message"`
}
type anthropicSSEMessage struct {
	ID      string                   `json:"id"`
	Type    string                   `json:"type"`
	Role    string                   `json:"role"`
	Content []anthropicContentBlock  `json:"content"`
	Model   string                   `json:"model"`
	Usage   anthropicSSEUsage        `json:"usage"`
}
type anthropicSSEUsage struct {
	InputTokens int `json:"input_tokens"`
}
type anthropicSSEContentStart struct {
	Type        string                  `json:"type"`
	Index       int                     `json:"index"`
	ContentBlock anthropicContentBlock  `json:"content_block"`
}
type anthropicSSEDelta struct {
	Type  string           `json:"type"`
	Index int              `json:"index"`
	Delta anthropicSSEText `json:"delta"`
}
type anthropicSSEText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type anthropicSSEContentStop struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}
type anthropicSSEMsgDelta struct {
	Type  string              `json:"type"`
	Delta anthropicSSEStop    `json:"delta"`
	Usage anthropicSSEOutUsage `json:"usage"`
}
type anthropicSSEStop struct {
	StopReason   string `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}
type anthropicSSEOutUsage struct {
	OutputTokens int `json:"output_tokens"`
}
type anthropicSSEMsgStop struct {
	Type string `json:"type"`
}

func writeSSE(w gin.ResponseWriter, event string, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonBytes))
}

// ── Streaming handler for Anthropic /v1/messages ──────────────────────────

func handleAnthropicStream(c *gin.Context, resp *http.Response, user *model.User,
	token *model.Token, channel *model.Channel, modelName string, startTime time.Time) {

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")

	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	var promptTokens, completionTokens int
	var contentStarted, messageStarted bool

	writer := c.Writer
	flusher, canFlush := writer.(http.Flusher)
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if !messageStarted {
			writeSSE(writer, "message_start", anthropicSSEMsgStart{
				Type: "message_start",
				Message: anthropicSSEMessage{
					ID:      msgID,
					Type:    "message",
					Role:    "assistant",
					Content: []anthropicContentBlock{},
					Model:   modelName,
					Usage:   anthropicSSEUsage{InputTokens: 0},
				},
			})
			if canFlush { flusher.Flush() }
			messageStarted = true
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" && !contentStarted {
				writeSSE(writer, "content_block_start", anthropicSSEContentStart{
					Type:        "content_block_start",
					Index:       0,
					ContentBlock: anthropicContentBlock{Type: "text", Text: ""},
				})
				if canFlush { flusher.Flush() }
				contentStarted = true
			}
			if delta.Content != "" {
				writeSSE(writer, "content_block_delta", anthropicSSEDelta{
					Type:  "content_block_delta",
					Index: 0,
					Delta: anthropicSSEText{Type: "text_delta", Text: delta.Content},
				})
				if canFlush { flusher.Flush() }
			}
			if chunk.Choices[0].FinishReason != "" {
				writeSSE(writer, "content_block_stop", anthropicSSEContentStop{
					Type: "content_block_stop", Index: 0,
				})
				writeSSE(writer, "message_delta", anthropicSSEMsgDelta{
					Type: "message_delta",
					Delta: anthropicSSEStop{StopReason: mapStopReason(chunk.Choices[0].FinishReason), StopSequence: nil},
					Usage: anthropicSSEOutUsage{OutputTokens: completionTokens},
				})
				writeSSE(writer, "message_stop", anthropicSSEMsgStop{Type: "message_stop"})
				if canFlush { flusher.Flush() }
			}
		}

		if chunk.Usage.PromptTokens > 0 {
			promptTokens = chunk.Usage.PromptTokens
		}
		if chunk.Usage.CompletionTokens > 0 {
			completionTokens = chunk.Usage.CompletionTokens
		}
	}

	if !messageStarted {
		writeSSE(writer, "message_start", anthropicSSEMsgStart{
			Type: "message_start",
			Message: anthropicSSEMessage{
				ID: msgID, Type: "message", Role: "assistant",
				Content: []anthropicContentBlock{},
				Model:   modelName,
				Usage:   anthropicSSEUsage{InputTokens: 0},
			},
		})
		writeSSE(writer, "message_delta", anthropicSSEMsgDelta{
			Type: "message_delta",
			Delta: anthropicSSEStop{StopReason: "end_turn", StopSequence: nil},
			Usage: anthropicSSEOutUsage{OutputTokens: 0},
		})
		writeSSE(writer, "message_stop", anthropicSSEMsgStop{Type: "message_stop"})
		if canFlush { flusher.Flush() }
	}

	writer.Write([]byte("\n"))
	if canFlush { flusher.Flush() }

	if c.GetString("idempotencyKey") != "" {
		idemCache.set(c.GetString("idempotencyKey"), &idempotencyEntry{createdAt: time.Now()})
	}

	go recordLog(user, token, channel, modelName, promptTokens, completionTokens, 0,
		resp.StatusCode, c.Param("path"), startTime)
}
