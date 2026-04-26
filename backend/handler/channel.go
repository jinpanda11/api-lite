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

// ListChannels godoc
// GET /api/channel
func ListChannels(c *gin.Context) {
	var channels []model.Channel
	if err := model.DB.Order("priority desc").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

// CreateChannel godoc
// POST /api/channel
func CreateChannel(c *gin.Context) {
	var req model.Channel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := model.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
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

	var req model.Channel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = channel.ID
	if err := model.DB.Model(&channel).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
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

	// Test connectivity by calling a suitable endpoint on the upstream
	baseURL := strings.TrimRight(channel.BaseURL, "/")
	var testMethod, testURL string
	isImage := channel.Type == "image"

	if isImage {
		// Image APIs expect POST; test the actual endpoint
		if channel.FixedPath != "" {
			testURL = baseURL + channel.FixedPath
		} else {
			testURL = baseURL + "/v1/images/generations"
		}
		testMethod = "POST"
	} else {
		testURL = baseURL + "/models"
		if !strings.Contains(baseURL, "/v1") {
			testURL = baseURL + "/v1/models"
		}
		testMethod = "GET"
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var testReq *http.Request
	var err error
	if testMethod == "POST" {
		testReq, err = http.NewRequest("POST", testURL, strings.NewReader(`{}`))
	} else {
		testReq, err = http.NewRequest("GET", testURL, nil)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request: " + err.Error()})
		return
	}
	testReq.Header.Set("Authorization", "Bearer "+channel.APIKey)
	testReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(testReq)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "connection failed: " + err.Error(),
			"channel": channel.Name,
			"url":     testURL,
			"elapsed": elapsed,
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "..."
	}

	if resp.StatusCode >= 400 && !isImage {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":      "upstream returned " + strconv.Itoa(resp.StatusCode),
			"channel":    channel.Name,
			"url":        testURL,
			"status":     resp.StatusCode,
			"response":   bodyStr,
			"elapsed_ms": elapsed,
		})
		return
	}

	// For image channels, any response (even 4xx) means the server is reachable.
	// A 401/422 is a valid response that proves connectivity.
	msg := "connection ok"
	if isImage && resp.StatusCode >= 400 {
		msg = fmt.Sprintf("connection ok (upstream returned %d, which is normal for image APIs)", resp.StatusCode)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    msg,
		"channel":    channel.Name,
		"url":        testURL,
		"status":     resp.StatusCode,
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
				inP, outP, bm, cp := getModelPricing(m)
				models = append(models, ModelInfo{
					ID:          m,
					ChannelName: ch.Name,
					InputPrice:  inP,
					OutputPrice: outP,
					BillingMode: bm,
					CallPrice:   cp,
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
					inP, outP, bm, cp := getModelPricing(m)
					models = append(models, ModelInfo{
						ID:          m,
						ChannelName: ch.Name,
						InputPrice:  inP,
						OutputPrice: outP,
						BillingMode: bm,
						CallPrice:   cp,
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
func getModelPricing(modelName string) (inputPrice, outputPrice float64, billingMode string, callPrice float64) {
	var mp model.ModelPricing
	if err := model.DB.Where("model_name = ?", modelName).First(&mp).Error; err == nil {
		billingMode = mp.BillingMode
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
// GET /v1/models — uses API key auth (same as relay) so CherryStudio etc can list models.
func OpenAIModelsList(c *gin.Context) {
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

	dbToken, err := model.GetTokenByKey(parts[1])
	if err != nil {
		c.JSON(http.StatusUnauthorized, openAIError("invalid or expired API key"))
		return
	}

	user, err := model.GetUserByID(dbToken.UserID)
	if err != nil || user.Status != model.StatusEnabled {
		c.JSON(http.StatusUnauthorized, openAIError("user not found or disabled"))
		return
	}

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
			inP, outP, bm, cp := getModelPricing(m)
			result = append(result, gin.H{
				"id":           m,
				"object":       "model",
				"created":      createdAt,
				"owned_by":     ch.Name,
				"input_price":  inP,
				"output_price": outP,
				"billing_mode": bm,
				"call_price":   cp,
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
