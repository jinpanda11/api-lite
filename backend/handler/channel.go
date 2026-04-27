package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"new-api-lite/model"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)


// channelRequest limits which fields can be set during create/update.
type channelRequest struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	Models    string `json:"models"`
	Priority  int    `json:"priority"`
	FixedPath string `json:"fixed_path"`
}

// ListChannels godoc
// GET /api/channel
func ListChannels(c *gin.Context) {
	var channels []model.Channel
	if err := model.DB.Order("priority desc").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": maskChannels(channels)})
}

// CreateChannel godoc
// POST /api/channel
func CreateChannel(c *gin.Context) {
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	ch := model.Channel{
		Name:      req.Name,
		Type:      req.Type,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		Models:    req.Models,
		Priority:  req.Priority,
		FixedPath: req.FixedPath,
		Status:    1,
	}
	if err := model.DB.Create(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}
	ch.APIKey = maskAPIKey(ch.APIKey)
	c.JSON(http.StatusOK, gin.H{"data": ch})
}

// UpdateChannel godoc
// PUT /api/channel/:id
func UpdateChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var channel model.Channel
	if err := model.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	updates := map[string]interface{}{
		"name":       req.Name,
		"type":       req.Type,
		"base_url":   req.BaseURL,
		"models":     req.Models,
		"priority":   req.Priority,
		"fixed_path": req.FixedPath,
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if err := model.DB.Model(&channel).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}
	channel.APIKey = maskAPIKey(channel.APIKey)
	c.JSON(http.StatusOK, gin.H{"data": channel})
}

// DeleteChannel godoc
// DELETE /api/channel/:id
func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.DB.Delete(&model.Channel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// TestChannel godoc
// POST /api/channel/test  { id: 1 }
func TestChannel(c *gin.Context) {
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var channel model.Channel
	if err := model.DB.First(&channel, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Test connectivity: try /v1/models first, fall back to /v1/chat/completions
	baseURL := strings.TrimRight(channel.BaseURL, "/")
	var testMethod, testURL, fallbackURL string
	isImage := channel.Type == "image"

	if isImage {
		if channel.FixedPath != "" {
			testURL = baseURL + channel.FixedPath
		} else {
			testURL = baseURL + "/v1/images/generations"
		}
		testMethod = "POST"
	} else {
		testURL = baseURL + "/v1/models"
		fallbackURL = baseURL + "/v1/chat/completions"
		testMethod = "GET"
	}

	client := &http.Client{Timeout: 15 * time.Second}

	doTest := func(url, method string, body io.Reader) (int, string, int64, error) {
		var req *http.Request
		var err error
		if method == "POST" {
			req, err = http.NewRequest("POST", url, body)
		} else {
			req, err = http.NewRequest("GET", url, nil)
		}
		if err != nil {
			return 0, "", 0, err
		}
		req.Header.Set("Authorization", "Bearer "+channel.APIKey)
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return 0, "", elapsed, err
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return resp.StatusCode, bodyStr, elapsed, nil
	}

	statusCode, bodyStr, elapsed, err := doTest(testURL, testMethod, strings.NewReader(`{}`))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "connection failed: " + err.Error(),
			"channel": channel.Name,
			"url":     testURL,
			"elapsed": elapsed,
		})
		return
	}

	// If primary test returned 404 and we have a fallback, retry with chat completions
	usedFallback := false
	if statusCode == 404 && fallbackURL != "" {
		fallbackBody := strings.NewReader(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"hi"}],"max_tokens":1}`)
		fbStatus, fbBody, fbElapsed, fbErr := doTest(fallbackURL, "POST", fallbackBody)
		if fbErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "connection failed (fallback): " + fbErr.Error(),
				"channel": channel.Name,
				"url":     fallbackURL,
				"elapsed": fbElapsed,
			})
			return
		}
		statusCode = fbStatus
		bodyStr = fbBody
		elapsed = fbElapsed
		usedFallback = true
	}

	if statusCode >= 400 && !isImage {
		errMsg := "upstream returned " + strconv.Itoa(statusCode)
		if usedFallback {
			errMsg = "upstream returned " + strconv.Itoa(statusCode) + " (tried /v1/models and /v1/chat/completions)"
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"error":      errMsg,
			"channel":    channel.Name,
			"url":        testURL,
			"status":     statusCode,
			"response":   bodyStr,
			"elapsed_ms": elapsed,
		})
		return
	}

	// For image channels, any response (even 4xx) means the server is reachable.
	msg := "connection ok"
	if usedFallback {
		msg = "connection ok (via /v1/chat/completions fallback; /v1/models not available)"
	}
	if isImage && statusCode >= 400 {
		if statusCode == 404 && channel.FixedPath == "" {
			msg = "server reachable, but /v1/images/generations not found — set a fixed path for this channel"
		} else {
			msg = fmt.Sprintf("connection ok (upstream returned %d)", statusCode)
		}
	}

	displayURL := testURL
	if usedFallback {
		displayURL = fallbackURL
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    msg,
		"channel":    channel.Name,
		"url":        displayURL,
		"status":     statusCode,
		"elapsed_ms": elapsed,
		"models":     bodyStr,
	})
}

// ListModels godoc
// GET /api/models — aggregated model list from all enabled channels
func ListModels(c *gin.Context) {
	channels, err := model.GetAvailableChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type ModelInfo struct {
		ID           string  `json:"id"`
		ChannelName  string  `json:"channel_name"`
		InputPrice   float64 `json:"input_price"`
		OutputPrice  float64 `json:"output_price"`
		BillingMode  string  `json:"billing_mode,omitempty"`
		CallPrice    float64 `json:"call_price,omitempty"`
		IconURL      string  `json:"icon_url,omitempty"`
	}

	// Fetch upstream models for channels with empty model list
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		models []ModelInfo
	)
	seen := sync.Map{}

	for _, ch := range channels {
		if ch.Models != "" {
			// Use manually configured model list
			for _, m := range splitModels(ch.Models) {
				if m == "" {
					continue
				}
				key := m + "|" + ch.Name
				if _, loaded := seen.LoadOrStore(key, true); loaded {
					continue
				}
				inP, outP, bm, cp, icon := getModelPricing(m)
				models = append(models, ModelInfo{
					ID:          m,
					ChannelName: ch.Name,
					InputPrice:  inP,
					OutputPrice: outP,
					BillingMode: bm,
					CallPrice:   cp,
				IconURL:     icon,
				})
			}
		} else {
			// Fetch model list from upstream API
			wg.Add(1)
			go func(ch model.Channel) {
				defer wg.Done()
				upstreamModels := fetchUpstreamModels(ch)
				mu.Lock()
				for _, m := range upstreamModels {
					key := m + "|" + ch.Name
					if _, loaded := seen.LoadOrStore(key, true); loaded {
						continue
					}
					inP, outP, bm, cp, icon := getModelPricing(m)
					models = append(models, ModelInfo{
						ID:          m,
						ChannelName: ch.Name,
						InputPrice:  inP,
						OutputPrice: outP,
						BillingMode: bm,
						CallPrice:   cp,
						IconURL:     icon,
					})
				}
				mu.Unlock()
			}(ch)
		}
	}
	wg.Wait()

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	c.JSON(http.StatusOK, gin.H{"data": models})
}

// fetchUpstreamModels calls GET /v1/models on the upstream and returns model IDs.
func fetchUpstreamModels(ch model.Channel) []string {
	baseURL := strings.TrimRight(ch.BaseURL, "/")
	modelsURL := baseURL + "/models"
	if !strings.Contains(baseURL, "/v1") {
		modelsURL = baseURL + "/v1/models"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		fmt.Printf("[MODELS] failed to build request for %s: %v\n", ch.Name, err)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+ch.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[MODELS] failed to fetch models from %s: %v\n", ch.Name, err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("[MODELS] failed to parse models from %s: %v\n", ch.Name, err)
		return nil
	}

	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// getModelPricing returns effective pricing for a model from global ModelPricing table.
func getModelPricing(modelName string) (inputPrice, outputPrice float64, billingMode string, callPrice float64, iconURL string) {
	var mp model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&mp).Error; err == nil {
		billingMode = mp.BillingMode
		iconURL = mp.IconURL
		if billingMode == "call" {
			callPrice = mp.CallPrice
		} else {
			inputPrice = mp.InputPrice
			outputPrice = mp.OutputPrice
		}
	}
	return
}

func splitModels(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trim(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

// maskAPIKey shows only first 4 and last 4 characters of an API key.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func maskChannels(channels []model.Channel) []model.Channel {
	out := make([]model.Channel, len(channels))
	for i, ch := range channels {
		ch.APIKey = maskAPIKey(ch.APIKey)
		out[i] = ch
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// OpenAIModelsList returns the model list in OpenAI-compatible format.
// GET /v1/models — auth is optional; when called from Relay the caller has already authenticated.
func OpenAIModelsList(c *gin.Context) {
	channels, err := model.GetAvailableChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, openAIError("failed to load channels"))
		return
	}

	const createdAt = 1626777600
	var result []gin.H

	for _, ch := range channels {
		var modelIDs []string
		if ch.Models != "" {
			modelIDs = splitModels(ch.Models)
		} else {
			modelIDs = fetchUpstreamModels(ch)
		}
		for _, m := range modelIDs {
			if m == "" {
				continue
			}
			inP, outP, bm, cp, icon := getModelPricing(m)
			result = append(result, gin.H{
				"id":           m,
				"object":       "model",
				"created":      createdAt,
				"owned_by":     ch.Name,
				"input_price":  inP,
				"output_price": outP,
				"billing_mode": bm,
				"call_price":   cp,
				"icon_url":    icon,
			})
		}
	}

	if result == nil {
		result = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   result,
	})
}
