package handler

import (
	"fmt"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"new-api-lite/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Relay is the core proxy handler for all /v1/* endpoints.
// Flow: validate token → select channel → forward request → record log → return response
func Relay(c *gin.Context) {
	// ── Handle /v1/models without auth (connectivity checks, model discovery) ──
	if c.Param("path") == "/models" {
		OpenAIModelsList(c)
		return
	}

	// ── 1. Extract and validate the API token ──────────────────────────────────
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, openAIError("missing Authorization header"))
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, openAIError("invalid Authorization header format"))
		return
	}
	apiKey := parts[1]

	dbToken, err := model.GetTokenByKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, openAIError("invalid or expired API key"))
		return
	}

	user, err := model.GetUserByID(dbToken.UserID)
	if err != nil || user.Status != model.StatusEnabled {
		c.JSON(http.StatusUnauthorized, openAIError("user not found or disabled"))
		return
	}

	// ── Handle /v1/models locally (avoid upstream relay) ─────────────────────
	if c.Param("path") == "/models" {
		OpenAIModelsList(c)
		return
	}

	// ── Check balance ──────────────────────────────────────────────────────────────
	if user.Balance <= 0 {
		c.JSON(http.StatusPaymentRequired, openAIError("insufficient balance, please top up"))
		return
	}

	// ── 2. Read and parse request body ──────────────────────────────────────────
	const maxBody = 10 << 20 // 10 MB
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody+1))
	if len(bodyBytes) > maxBody {
		c.JSON(http.StatusRequestEntityTooLarge, openAIError("request body too large"))
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, openAIError("failed to read request body"))
		return
	}

	// Extract model name for channel selection and billing
	modelName := extractModel(bodyBytes, c.Param("path"))
	isStream := isStreamRequest(bodyBytes)

	// ── 3. Select upstream channel ───────────────────────────────────────────────
	channel, err := model.SelectChannel(modelName)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, openAIError("no available channel for model: "+modelName))
		return
	}

	// ── 4. Build upstream request ────────────────────────────────────────────────
	// Strip /v1 from base URL (if present) then append the full request path,
	// so both "https://api.openai.com" and "https://api.openai.com/v1" work.
	var upstreamURL string
	if channel.Type == "image" && channel.FixedPath != "" {
		upstreamURL = strings.TrimRight(channel.BaseURL, "/") + channel.FixedPath
		if c.Request.URL.RawQuery != "" {
			upstreamURL += "?" + c.Request.URL.RawQuery
		}
	} else {
		baseURL := strings.TrimRight(channel.BaseURL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/v1")
		upstreamURL = baseURL + c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			upstreamURL += "?" + c.Request.URL.RawQuery
		}
	}

	req, err := http.NewRequest(c.Request.Method, upstreamURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, openAIError("failed to build upstream request"))
		return
	}

	// Copy all original headers, then override Authorization with channel key.
	// Skip hop-by-hop headers that must not be forwarded.
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
	}
	for k, vs := range c.Request.Header {
		if hopByHop[strings.ToLower(k)] {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// ── 5. Execute upstream request ──────────────────────────────────────────────
	startTime := time.Now()
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, openAIError("upstream request failed: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	// ── 6. Copy response headers to client ───────────────────────────────────────

	isImageChannel := channel.Type == "image"
		contentType := resp.Header.Get("Content-Type")
	isSSE := strings.Contains(contentType, "text/event-stream")

		if isImageChannel && !isSSE {
			// Peek at body to detect SSE even with wrong Content-Type
		if !isSSE {
			bufReader := bufio.NewReader(resp.Body)
			peek, _ := bufReader.Peek(6)
			isSSE = strings.HasPrefix(string(peek), "data: ")
			resp.Body = io.NopCloser(bufReader)
		}
		}
	for k, vs := range resp.Header {
		if isImageChannel && isSSE && strings.EqualFold(k, "Content-Type") {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}


	// ──── 7. Handle response (stream vs non-stream) ────────────────
	if isStream {
		handleStream(c, resp, user, dbToken, channel, modelName, startTime)
	} else if isImageChannel && isSSE {
		handleSSENonStream(c, resp, user, dbToken, channel, modelName, startTime)
	} else {
		handleNonStream(c, resp, user, dbToken, channel, modelName, startTime)
	}
}

// handleStream pipes SSE events back to the client and parses usage on completion.
func handleStream(c *gin.Context, resp *http.Response, user *model.User,
	token *model.Token, channel *model.Channel, modelName string, startTime time.Time) {

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	var promptTokens, completionTokens, cacheTokens int
	writer := c.Writer
	flusher, canFlush := writer.(http.Flusher)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := writer.Write([]byte(line + "\n")); err != nil {
			break
		}
		if canFlush {
			flusher.Flush()
		}

		// Parse usage from the final [DONE] data chunk or usage chunk
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
		}
			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk.Usage.PromptTokens > 0 {
					promptTokens = chunk.Usage.PromptTokens
					completionTokens = chunk.Usage.CompletionTokens
					if chunk.Usage.PromptTokensDetails != nil {
						cacheTokens = chunk.Usage.PromptTokensDetails.CachedTokens
					}
				}
		}
		}
	}

	// Write trailing newline to close the stream
	writer.Write([]byte("\n"))
	if canFlush {
		flusher.Flush()
	}

	// Record log asynchronously to avoid blocking the response
	go recordLog(user, token, channel, modelName, promptTokens, completionTokens, cacheTokens,
		resp.StatusCode, c.Param("path"), startTime)
}

// handleNonStream reads the full response, records log, and returns to client.
func handleNonStream(c *gin.Context, resp *http.Response, user *model.User,
	token *model.Token, channel *model.Channel, modelName string, startTime time.Time) {

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, openAIError("failed to read upstream response"))
		return
	}

	var promptTokens, completionTokens, cacheTokens int
	var cr completionResponse
	if json.Unmarshal(bodyBytes, &cr) == nil {
		promptTokens = cr.Usage.PromptTokens
		completionTokens = cr.Usage.CompletionTokens
		if cr.Usage.PromptTokensDetails != nil {
			cacheTokens = cr.Usage.PromptTokensDetails.CachedTokens
		}
	}

	go recordLog(user, token, channel, modelName, promptTokens, completionTokens, cacheTokens,
		resp.StatusCode, c.Param("path"), startTime)

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// recordLog deducts balance and writes an audit log entry.
func recordLog(user *model.User, token *model.Token, channel *model.Channel,
	modelName string, inputTokens, outputTokens, cacheTokens, statusCode int, path string, _ time.Time) {

	var inputPrice, outputPrice float64

	// Look up global model pricing
	var mp model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&mp).Error; err == nil {
		if mp.BillingMode == "call" {
			totalCost := mp.CallPrice
			status := 1
			if statusCode >= 400 {
				status = 2
				totalCost = 0
			}
			if totalCost > 0 {
				if !user.DeductBalance(totalCost) {
					totalCost = 0
					status = 2
				}
			}
			log := &model.Log{
				UserID:       user.ID,
				TokenID:      token.ID,
				TokenName:    token.Name,
				ChannelID:    channel.ID,
				ChannelName:  channel.Name,
				Model:        modelName,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				Cost:         totalCost,
				Status:       status,
				RequestPath:  path,
			}
			_ = model.CreateLog(log)
			return
		}
		// Token billing mode: use global prices
		inputPrice = mp.InputPrice
		outputPrice = mp.OutputPrice
	}

	inputCost := float64(inputTokens) / 1000000.0 * inputPrice
	outputCost := float64(outputTokens) / 1000000.0 * outputPrice
	cacheCost := float64(cacheTokens) / 1000000.0 * inputPrice // cache billed at input price
	totalCost := inputCost + outputCost + cacheCost

	status := 1
	if statusCode >= 400 {
		status = 2
		totalCost = 0
	}

	if totalCost > 0 {
		if !user.DeductBalance(totalCost) {
			totalCost = 0
			status = 2
		}
	}

	log := &model.Log{
		UserID:       user.ID,
		TokenID:      token.ID,
		TokenName:    token.Name,
		ChannelID:    channel.ID,
		ChannelName:  channel.Name,
		Model:        modelName,
		InputTokens:  inputTokens + cacheTokens,
		OutputTokens: outputTokens,
		Cost:         totalCost,
		Status:       status,
		RequestPath:  path,
	}
	_ = model.CreateLog(log)
}




// handleSSENonStream converts upstream SSE response to JSON for non-stream requests.
func handleSSENonStream(c *gin.Context, resp *http.Response, user *model.User,
	token *model.Token, channel *model.Channel, modelName string, startTime time.Time) {

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, openAIError("failed to read upstream response"))
		return
	}

	// Collect all SSE data events to find the final result
	var lastData string
	for _, line := range strings.Split(string(bodyBytes), "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "data: ") {
			lastData = strings.TrimPrefix(trimmed, "data: ")
		}
	}

	// Try to extract image URL from the final event (grsai format: results[].url)
	if lastData != "" {
		var event struct {
			Results []struct {
				URL string `json:"url"`
		} `json:"results"`
		}
		if json.Unmarshal([]byte(lastData), &event) == nil && len(event.Results) > 0 {
			urls := make([]gin.H, len(event.Results))
			for i, r := range event.Results {
				urls[i] = gin.H{"url": r.URL, "b64_json": nil}
		}
			// Determine response format based on original request path
			path := c.Param("path")
			isChatEndpoint := strings.Contains(path, "/chat/completions")

			if isChatEndpoint {
				// Return chat completions format for CherryStudio etc.
				c.JSON(http.StatusOK, gin.H{
					"choices": []gin.H{
						{
							"message": gin.H{
								"content": fmt.Sprintf("![image](%s)", event.Results[0].URL),
								"role":     "assistant",
							},
							"finish_reason": "stop",
						},
					},
					"usage": gin.H{
						"prompt_tokens":     0,
						"completion_tokens": 0,
						"total_tokens":      0,
					},
					"created": time.Now().Unix(),
					"model":   modelName,
				})
		} else {
				// Return OpenAI image generation format for /v1/images/generations
				c.JSON(http.StatusOK, gin.H{
					"created": time.Now().Unix(),
					"data":    urls,
				})
		}
			go recordLog(user, token, channel, modelName, 0, 0, 0,
				http.StatusOK, c.Param("path"), startTime)
			return
	}

	// Fallback: upstream returned SSE format we cannot parse
	c.Data(resp.StatusCode, "text/plain; charset=utf-8", bodyBytes)
	go recordLog(user, token, channel, modelName, 0, 0, 0,
		resp.StatusCode, c.Param("path"), startTime)
}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

type completionResponse struct {
	Usage struct {
		PromptTokens          int `json:"prompt_tokens"`
		CompletionTokens      int `json:"completion_tokens"`
		PromptTokensDetails   *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details,omitempty"`
	} `json:"usage"`
}

type streamChunk struct {
	Usage struct {
		PromptTokens          int `json:"prompt_tokens"`
		CompletionTokens      int `json:"completion_tokens"`
		PromptTokensDetails   *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details,omitempty"`
	} `json:"usage"`
}

func extractModel(body []byte, path string) string {
	// Try to parse from JSON body first
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err == nil && req.Model != "" {
		return req.Model
	}
	// Fall back to path hint (e.g. /v1/models doesn't have a body)
	if strings.Contains(path, "chat/completions") {
		return "gpt-3.5-turbo"
	}
	return ""
}

func isStreamRequest(body []byte) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err == nil {
		return req.Stream
	}
	return false
}

func openAIError(msg string) gin.H {
	return gin.H{
		"error": gin.H{
			"message": msg,
			"type":    "api_error",
			"code":    nil,
		},
	}
}
