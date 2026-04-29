package handler

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"new-api-lite/middleware"
	"new-api-lite/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func shortUUID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── Session ──────────────────────────────────────────────────────────────────

func ChatSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "Success",
		"data": gin.H{
			"auth":     false,
			"model":    "ChatGPTAPI",
			"amodel":   "gpt-3.5-turbo",
			"isWsrv":   false,
			"isUpload": false,
			"theme":    "light",
			"models":   gatherModelNames(),
		},
	})
}

// gatherModelNames returns unique model names from model_pricings and channels.
func gatherModelNames() []string {
	seen := map[string]bool{}

	var pricings []model.ModelPricing
	if err := model.DB.Find(&pricings).Error; err == nil {
		for _, p := range pricings {
			seen[p.ModelName] = true
		}
	}

	var channels []model.Channel
	if err := model.DB.Find(&channels).Error; err == nil {
		for _, ch := range channels {
			for _, m := range strings.Split(ch.Models, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					seen[m] = true
				}
			}
		}
	}

	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}

// ── Config ───────────────────────────────────────────────────────────────────

func ChatConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "Success",
		"message": "",
		"data":    gin.H{},
	})
}

// ── Chat Process ─────────────────────────────────────────────────────────────

type ChatProcessRequest struct {
	Prompt        string       `json:"prompt"`
	Options       *ChatOptions `json:"options"`
	SystemMessage string       `json:"systemMessage"`
	Temperature   *float64     `json:"temperature"`
	TopP          *float64     `json:"top_p"`
	Model         string       `json:"model"`
}

type ChatOptions struct {
	ConversationID  string `json:"conversationId"`
	ParentMessageID string `json:"parentMessageId"`
}

func ChatProcess(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	var req ChatProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	// Build OpenAI chat completions request
	modelName := req.Model
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}
	messages := []map[string]string{}
	if req.SystemMessage != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.SystemMessage})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.Prompt})

	openaiReq := map[string]interface{}{
		"model":    modelName,
		"messages": messages,
		"stream":   true,
	}
	if req.Temperature != nil {
		openaiReq["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		openaiReq["top_p"] = *req.TopP
	}

	bodyBytes, _ := json.Marshal(openaiReq)

	// Pre-consume billing
	estimatedCost, preDeducted := preConsume(user, modelName, bodyBytes)
	c.Set("preDeducted", preDeducted)
	if preDeducted == -1 {
		c.JSON(http.StatusPaymentRequired, openAIError(fmt.Sprintf(
			"insufficient balance (estimated: $%.4f, balance: $%.4f)", estimatedCost, user.Balance)))
		return
	}

	// Select channel and forward
	channels, err := model.SelectChannels(modelName)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, openAIError("no available channel"))
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}
	var resp *http.Response
	var usedChannel *model.Channel
	startTime := time.Now()

	for i := range channels {
		ch := &channels[i]
		upstreamURL := buildUpstreamURL(ch, "/v1/chat/completions", "", false)
		upstreamReq, _ := http.NewRequest("POST", upstreamURL, bytes.NewBuffer(bodyBytes))
		upstreamReq.Header.Set("Authorization", "Bearer "+ch.APIKey)
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(upstreamReq)
		if err != nil || (resp != nil && resp.StatusCode >= 500 && i < len(channels)-1) {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		usedChannel = ch
		break
	}

	if resp == nil {
		c.JSON(http.StatusBadGateway, openAIError("upstream request failed"))
		return
	}
	defer resp.Body.Close()

	// Collect response for fallback if upstream returns non-streaming
	bodyBuf := &bytes.Buffer{}
	teeReader := io.TeeReader(resp.Body, bodyBuf)

	// Build chat message IDs
	msgID := shortUUID()
	convID := shortUUID()
	if req.Options != nil && req.Options.ConversationID != "" {
		convID = req.Options.ConversationID
	}
	parentMsgID := ""
	if req.Options != nil {
		parentMsgID = req.Options.ParentMessageID
	}

	// Set streaming response headers
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	writer := c.Writer
	flusher, _ := writer.(http.Flusher)
	firstChunk := true
	var fullText strings.Builder
	var promptTokens, completionTokens int

	scanner := bufio.NewScanner(teeReader)
	scanBuf := make([]byte, 0, 64*1024)
	scanner.Buffer(scanBuf, 1024*1024)

	msgSent := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var sseChunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &sseChunk); err != nil {
			continue
		}
		if len(sseChunk.Choices) > 0 {
			fullText.WriteString(sseChunk.Choices[0].Delta.Content)
		}
		if sseChunk.Usage != nil {
			promptTokens = sseChunk.Usage.PromptTokens
			completionTokens = sseChunk.Usage.CompletionTokens
		}

		msg := gin.H{
			"id":              msgID,
			"text":            fullText.String(),
			"role":            "assistant",
			"conversationId":  convID,
			"parentMessageId": parentMsgID,
		}

		msgJSON, _ := json.Marshal(msg)
		if firstChunk {
			writer.Write(msgJSON)
			firstChunk = false
		} else {
			writer.Write([]byte("\n"))
			writer.Write(msgJSON)
		}
		msgSent = true

		if flusher != nil {
			flusher.Flush()
		}
	}

	// Non-streaming fallback
	if !msgSent {
		var nonStreamResp map[string]interface{}
		if json.NewDecoder(bodyBuf).Decode(&nonStreamResp) == nil {
			content := ""
			if choices, ok := nonStreamResp["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if m, ok := choice["message"].(map[string]interface{}); ok {
						if c, ok := m["content"].(string); ok {
							content = c
						}
					}
				}
			}
			if usage, ok := nonStreamResp["usage"].(map[string]interface{}); ok {
				if pt, ok := usage["prompt_tokens"].(float64); ok {
					promptTokens = int(pt)
				}
				if ct, ok := usage["completion_tokens"].(float64); ok {
					completionTokens = int(ct)
				}
			}

			msg := gin.H{
				"id":              msgID,
				"text":            content,
				"role":            "assistant",
				"conversationId":  convID,
				"parentMessageId": parentMsgID,
			}
			msgJSON, _ := json.Marshal(msg)
			writer.Write(msgJSON)
			msgSent = true
		}
	}

	if flusher != nil {
		flusher.Flush()
	}

	// Log billing
	tokenName := "chat-web"
	if tokens, err := model.GetTokensByUserID(user.ID); err == nil && len(tokens) > 0 {
		tokenName = tokens[0].Name
	}

	syntheticToken := &model.Token{Name: tokenName}
	enqueueLog(logTask{
		user:            user,
		token:           syntheticToken,
		channel:         usedChannel,
		modelName:       modelName,
		inputTokens:     promptTokens,
		outputTokens:    completionTokens,
		statusCode:      http.StatusOK,
		path:            "/chat-process",
		preDeducted:     preDeducted,
		startTime:       startTime,
		checkThresholds: true,
	})
}
