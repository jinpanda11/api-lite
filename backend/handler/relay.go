package handler

import (
	"encoding/base64"
	"fmt"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"new-api-lite/model"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Idempotency cache ──────────────────────────────────────────────────────

type idempotencyEntry struct {
	statusCode  int
	contentType string
	body        []byte
	createdAt   time.Time
}

type idempotencyCacheStruct struct {
	mu    sync.RWMutex
	items map[string]*idempotencyEntry
}

var idemCache = &idempotencyCacheStruct{
	items: make(map[string]*idempotencyEntry),
}

func init() {
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			idemCache.mu.Lock()
			now := time.Now()
			for k, v := range idemCache.items {
				if now.Sub(v.createdAt) > 5*time.Minute {
					delete(idemCache.items, k)
				}
			}
			idemCache.mu.Unlock()
		}
	}()
}

func (c *idempotencyCacheStruct) get(key string) *idempotencyEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Since(e.createdAt) > 5*time.Minute {
		return nil
	}
	return e
}

func (c *idempotencyCacheStruct) set(key string, e *idempotencyEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = e
}


// Relay is the core proxy handler for all /v1/* endpoints.
// Flow: validate token → select channel → forward request → record log → return response
func Relay(c *gin.Context) {
	// ── Handle /v1/models without auth (connectivity checks, model discovery) ──
	if c.Param("path") == "/models" {
		OpenAIModelsList(c)
		return
	}

	// ── 1. Extract and validate the API token ──────────────────────────────────
	// Accept both "Authorization: Bearer <key>" (OpenAI) and "x-api-key: <key>" (Anthropic)
	apiKey := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if apiKey == c.GetHeader("Authorization") { // no Bearer prefix found
		apiKey = c.GetHeader("x-api-key")
	}
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, openAIError("missing API key (use Authorization: Bearer or x-api-key)"))
		return
	}

	dbToken, err := model.CacheGetTokenByKey(apiKey)
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


	// ── 2. Read and parse request body ──────────────────────────────────────────
	const maxBody = 32 << 20 // 32 MB (accommodates image uploads)
	contentType := c.GetHeader("Content-Type")
	isMultipart := strings.HasPrefix(contentType, "multipart/form-data")

	var bodyBytes []byte
	var modelName string
	var isStream bool

	if isMultipart {
		bodyBytes, modelName, isStream, err = parseMultipartToJSON(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, openAIError("failed to parse multipart form: " + err.Error()))
			return
		}
	} else {
		bodyBytes, err = io.ReadAll(io.LimitReader(c.Request.Body, maxBody+1))
		if len(bodyBytes) > maxBody {
			c.JSON(http.StatusRequestEntityTooLarge, openAIError("request body too large"))
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, openAIError("failed to read request body"))
			return
		}

		// Extract model name for channel selection and billing
		modelName = extractModel(bodyBytes, c.Param("path"))
		isStream = isStreamRequest(bodyBytes)
	}

	// ── Anthropic format detection: /v1/messages → convert to OpenAI ────
	isAnthropicPath := c.Param("path") == "/messages"
	if isAnthropicPath {
		convertedBody, err := anthropicToOpenAI(bodyBytes)
		if err == nil {
			bodyBytes = convertedBody
			// Re-extract model from converted body
			modelName = extractModel(bodyBytes, "/chat/completions")
		}
	}

	// ── Pre-consume: estimate cost, pre-deduct for call billing ────────
	estimatedCost, preDeducted := preConsume(user, modelName, bodyBytes)
	c.Set("preDeducted", preDeducted)
	if preDeducted < 0 {
		c.JSON(http.StatusPaymentRequired, openAIError(fmt.Sprintf(
			"insufficient balance (estimated: $%.4f, balance: $%.4f)", estimatedCost, user.Balance)))
		return
	}

	// ── Idempotency check ───────────────────────────────────────────────────
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	c.Set("idempotencyKey", idempotencyKey)
	if c.GetString("idempotencyKey") != "" {
		if cached := idemCache.get(idempotencyKey); cached != nil {
			go func() { recordLog(user, dbToken, nil, modelName, 0, 0, 0, cached.statusCode, c.Param("path"), getPreDeducted(c), time.Now()); checkUsageThresholds(user) }()
			c.Data(cached.statusCode, cached.contentType, cached.body)
			return
		}
	}

	// ── 3. Select upstream channels (failover support) ───────────────────────────
	channels, err := model.SelectChannels(modelName)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, openAIError("no available channel for model: "+modelName))
		return
	}

	// ── 4. Build and execute upstream request with failover ──────────────────────
	// Iterate channels: try first, fall back to next on 5xx or connection error.
	// 4xx errors are NOT retried (client error, not upstream fault).
	startTime := time.Now()
	client := &http.Client{Timeout: 120 * time.Second}
	var resp *http.Response
	var usedChannel *model.Channel
	var lastErr error
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

	for i := range channels {
		ch := &channels[i]
		// Build upstream URL for this channel
		upstreamURL := buildUpstreamURL(ch, c.Request.URL.Path, c.Request.URL.RawQuery, isAnthropicPath)
		req, err := http.NewRequest(c.Request.Method, upstreamURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		// Copy headers for each attempt (req is per-channel)
		for k, vs := range c.Request.Header {
			if hopByHop[strings.ToLower(k)] {
				continue
			}
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
		req.Header.Set("Authorization", "Bearer "+ch.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			continue // connection error → try next channel
		}
		// 5xx: upstream error → try next channel. 4xx: client error → keep response.
		if resp.StatusCode >= 500 && i < len(channels)-1 {
			resp.Body.Close()
			lastErr = fmt.Errorf("upstream returned %d", resp.StatusCode)
			continue
		}
		usedChannel = ch
		break
	}

	if resp == nil {
		errMsg := "all channels exhausted"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		c.JSON(http.StatusBadGateway, openAIError("upstream request failed: "+errMsg))
		return
	}
	defer resp.Body.Close()
	channel := usedChannel

	// ── 6. Copy response headers to client ───────────────────────────────────────

	isImageChannel := channel.Type == "image"
		contentType = resp.Header.Get("Content-Type")
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
	if isAnthropicPath && isStream {
		handleAnthropicStream(c, resp, user, dbToken, channel, modelName, startTime)
	} else if isAnthropicPath && !isStream {
		handleAnthropicNonStream(c, resp, user, dbToken, channel, modelName, startTime)
	} else if isStream {
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

	// Mark idempotency key as used (stream can't be cached but we prevent re-processing)
	if c.GetString("idempotencyKey") != "" {
		idemCache.set(c.GetString("idempotencyKey"), &idempotencyEntry{createdAt: time.Now()})
	}

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
	go func() { recordLog(user, token, channel, modelName, promptTokens, completionTokens, cacheTokens,
		resp.StatusCode, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()
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

	if c.GetString("idempotencyKey") != "" {
		idemCache.set(c.GetString("idempotencyKey"), &idempotencyEntry{
			statusCode:  resp.StatusCode,
			contentType: resp.Header.Get("Content-Type"),
			body:        bodyBytes,
			createdAt:   time.Now(),
		})
	}
	go func() { recordLog(user, token, channel, modelName, promptTokens, completionTokens, cacheTokens,
		resp.StatusCode, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()

	// For image channels, sanitize responses that have data:null (upstream error format)
	if channel != nil && channel.Type == "image" {
		var sanitized map[string]any
		if json.Unmarshal(bodyBytes, &sanitized) == nil {
			if d, ok := sanitized["data"]; !ok || d == nil {
				sanitized["data"] = []any{}
			}
			if b, err := json.Marshal(sanitized); err == nil {
				bodyBytes = b
			}
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// recordLog deducts balance and writes an audit log entry.
// preDeducted is the amount already deducted during pre-consume (call billing only).
func recordLog(user *model.User, token *model.Token, channel *model.Channel,
	modelName string, inputTokens, outputTokens, cacheTokens, statusCode int, path string, preDeducted float64, _ time.Time) {

	channelID := uint(0)
	channelName := ""
	if channel != nil {
		channelID = channel.ID
		channelName = channel.Name
	}

	var inputPrice, outputPrice float64

	// Look up global model pricing
	var mp model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&mp).Error; err == nil {
		if mp.BillingMode == "call" {
			totalCost := mp.CallPrice * user.PriceMultiplier
			status := 1
			if statusCode >= 400 {
				status = 2
				totalCost = 0
			}
			// If already pre-deducted, skip deduction. On 4xx/5xx, refund the pre-deduct.
			if preDeducted > 0 && statusCode >= 400 {
				user.AddBalance(preDeducted)
				totalCost = 0
			} else if preDeducted <= 0 && totalCost > 0 {
				if !user.DeductBalance(totalCost) {
					totalCost = 0
					status = 2
				}
			}
			log := &model.Log{
				UserID:       user.ID,
				TokenID:      token.ID,
				TokenName:    token.Name,
				ChannelID:    channelID,
				ChannelName:  channelName,
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
	totalCost := (inputCost + outputCost + cacheCost) * user.PriceMultiplier

	status := 1
	if statusCode >= 400 {
		status = 2
		totalCost = 0
	}

	// Token billing: deduct after response. Refund pre-deduct on error.
	if preDeducted > 0 && statusCode >= 400 {
		user.AddBalance(preDeducted)
	} else if totalCost > 0 && preDeducted <= 0 {
		if !user.DeductBalance(totalCost) {
			totalCost = 0
			status = 2
		}
	}

	log := &model.Log{
		UserID:       user.ID,
		TokenID:      token.ID,
		TokenName:    token.Name,
		ChannelID:    channelID,
		ChannelName:  channelName,
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

	// Collect SSE data events. Prefer the last line with actual results (non-null),
	// falling back to the very last data line if none completed.
	var lastData string
	var lastCompleted string
	for _, line := range strings.Split(string(bodyBytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "data: ") {
			continue
		}
		data := strings.TrimPrefix(trimmed, "data: ")
		if data == "[DONE]" || data == "" {
			continue
		}
		lastData = data
		// Detect completed data: has non-null "results" or "data" array, or status=succeeded/finished
		var peek struct {
			Results any    `json:"results"`
			Data    any    `json:"data"`
			Status  string `json:"status"`
		}
		if json.Unmarshal([]byte(data), &peek) == nil {
			if peek.Results != nil || peek.Data != nil {
				lastCompleted = data
			}
		}
	}
	if lastCompleted != "" {
		lastData = lastCompleted
	}

	if lastData != "" {
		path := c.Param("path")
		isChatEndpoint := strings.Contains(path, "/chat/completions")

		// Try 1: grsai format {"results":[{"url":"..."}]}
		var grsai struct {
			Results []struct {
				URL string `json:"url"`
			} `json:"results"`
		}
		if json.Unmarshal([]byte(lastData), &grsai) == nil && len(grsai.Results) > 0 {
			urls := make([]gin.H, len(grsai.Results))
			for i, r := range grsai.Results {
				urls[i] = gin.H{"url": r.URL}
			}
			if isChatEndpoint {
				c.JSON(http.StatusOK, gin.H{
					"choices": []gin.H{{
						"message":       gin.H{"content": fmt.Sprintf("![image](%s)", grsai.Results[0].URL), "role": "assistant"},
						"finish_reason": "stop",
					}},
					"usage":   gin.H{"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
					"created": time.Now().Unix(),
					"model":   modelName,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{"created": time.Now().Unix(), "data": urls})
			}
			go func() { recordLog(user, token, channel, modelName, 0, 0, 0,
				http.StatusOK, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()
			return
		}

		// Try 2: standard OpenAI image format {"data":[{"url":"..."}]}
		var oai struct {
			Data []struct {
				URL     string `json:"url"`
				B64JSON string `json:"b64_json"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(lastData), &oai) == nil && len(oai.Data) > 0 {
			urls := make([]gin.H, len(oai.Data))
			for i, d := range oai.Data {
				item := gin.H{"url": d.URL}
				if d.B64JSON != "" {
					item["b64_json"] = d.B64JSON
				}
				urls[i] = item
			}
			c.JSON(http.StatusOK, gin.H{"created": time.Now().Unix(), "data": urls})
			go func() { recordLog(user, token, channel, modelName, 0, 0, 0,
				http.StatusOK, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()
			return
		}

		// Try 3: lastData is valid JSON with a "data" field — pass through
		var generic map[string]interface{}
		if json.Unmarshal([]byte(lastData), &generic) == nil {
			// Sanitize: if "data" is null/missing, replace with empty array
			if _, ok := generic["data"]; !ok || generic["data"] == nil {
				generic["data"] = []any{}
			}
			sanitized, _ := json.Marshal(generic)
			c.Data(resp.StatusCode, "application/json", sanitized)
			go func() { recordLog(user, token, channel, modelName, 0, 0, 0,
				resp.StatusCode, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()
			return
		}
	}

	// Fallback: return empty data array instead of raw SSE bytes
	c.JSON(http.StatusOK, gin.H{"created": time.Now().Unix(), "data": []gin.H{}})
	go func() { recordLog(user, token, channel, modelName, 0, 0, 0,
		http.StatusOK, c.Param("path"), getPreDeducted(c), startTime); checkUsageThresholds(user) }()
}


// parseMultipartToJSON converts a multipart/form-data request (used by /v1/images/edits)
// into a JSON body suitable for forwarding to the upstream API.
func parseMultipartToJSON(c *gin.Context) ([]byte, string, bool, error) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return nil, "", false, err
	}

	model := c.Request.FormValue("model")
	prompt := c.Request.FormValue("prompt")
	n := c.Request.FormValue("n")
	size := c.Request.FormValue("size")
	responseFormat := c.Request.FormValue("response_format")

	body := map[string]any{
		"model":  model,
		"prompt": prompt,
	}
	if n != "" {
		body["n"] = n
	}
	if size != "" {
		body["size"] = size
	}
	if responseFormat != "" {
		body["response_format"] = responseFormat
	}

	// image field: try file upload first, fall back to form value (URL string)
	imageFile, header, _ := c.Request.FormFile("image")
	if imageFile != nil {
		imageData, err := io.ReadAll(imageFile)
		imageFile.Close()
		if err == nil && len(imageData) > 0 {
			mimeType := header.Header.Get("Content-Type")
			if mimeType == "" {
				mimeType = http.DetectContentType(imageData)
			}
			body["image"] = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imageData)
		}
	} else if imageURL := c.Request.FormValue("image"); imageURL != "" {
		body["image"] = imageURL
	}

	// mask field: optional mask image for inpainting
	maskFile, maskHeader, _ := c.Request.FormFile("mask")
	if maskFile != nil {
		maskData, err := io.ReadAll(maskFile)
		maskFile.Close()
		if err == nil && len(maskData) > 0 {
			mimeType := maskHeader.Header.Get("Content-Type")
			if mimeType == "" {
				mimeType = http.DetectContentType(maskData)
			}
			body["mask"] = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(maskData)
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, "", false, err
	}

	return jsonBody, model, false, nil
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

// getPreDeducted safely extracts the pre-deducted amount from gin context.
func getPreDeducted(c *gin.Context) float64 {
	v, ok := c.Get("preDeducted")
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// preConsume estimates the cost of a request and pre-deducts for call billing.
// Returns (estimatedCost, preDeducted).
// preDeducted == -1 means insufficient balance → reject the request.
func preConsume(user *model.User, modelName string, body []byte) (float64, float64) {
	var mp model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&mp).Error; err != nil {
		// Unknown model — allow through, billing will happen post-hoc if pricing exists
		return 0, 0
	}

	// Call billing: exact price known, pre-deduct atomically
	if mp.BillingMode == "call" {
		cost := mp.CallPrice * user.PriceMultiplier
		if cost <= 0 {
			return 0, 0
		}
		if !user.DeductBalance(cost) {
			return cost, -1
		}
		return cost, cost
	}

	// Token billing: estimate from body, check minimum balance
	estimatedInput := float64(len(body)) / 4.0
	estimatedOutput := estimatedInput * 0.3
	if estimatedOutput < 50 {
		estimatedOutput = 50
	}
	inputCost := estimatedInput / 1000000.0 * mp.InputPrice
	outputCost := estimatedOutput / 1000000.0 * mp.OutputPrice
	estimated := (inputCost + outputCost) * user.PriceMultiplier

	if estimated > 0 && user.Balance < estimated {
		return estimated, -1
	}
	return estimated, 0
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

func buildUpstreamURL(ch *model.Channel, path, rawQuery string, isAnthropicPath bool) string {
	if ch.Type == "image" && ch.FixedPath != "" {
		u := strings.TrimRight(ch.BaseURL, "/") + ch.FixedPath
		if rawQuery != "" {
			u += "?" + rawQuery
		}
		return u
	}
	baseURL := strings.TrimRight(ch.BaseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	var u string
	if isAnthropicPath {
		u = baseURL + "/v1/chat/completions"
	} else {
		u = baseURL + path
	}
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u
}
