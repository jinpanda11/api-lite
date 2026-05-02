package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"new-api-lite/middleware"
	"new-api-lite/model"

	"github.com/gin-gonic/gin"
)

type DrawRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
}

type DrawQuotaResponse struct {
	QuotaRemaining int  `json:"quota_remaining"`
	QuotaTotal     int  `json:"quota_total"`
	IsAdmin        bool `json:"is_admin"`
}

func GetDrawQuota(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	isAdmin := user.Role == model.RoleAdmin

	quotaTotal := model.DefaultDailyDrawQuota
	if v, err := model.GetSetting("daily_draw_quota"); err == nil && v != "" {
		if n, _ := fmt.Sscanf(v, "%d", &quotaTotal); n != 1 || quotaTotal <= 0 {
			quotaTotal = model.DefaultDailyDrawQuota
		}
	}

	remaining := user.GetDrawQuota()
	c.JSON(http.StatusOK, DrawQuotaResponse{
		QuotaRemaining: remaining,
		QuotaTotal:     quotaTotal,
		IsAdmin:        isAdmin,
	})
}

func DrawImage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	isAdmin := user.Role == model.RoleAdmin

	var req DrawRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	if len(req.Prompt) > 4096 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt too long (max 4096 chars)"})
		return
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gpt-image-2"
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}
	quality := req.Quality
	if quality == "" {
		quality = "standard"
	}

	// Quota check (admin bypasses)
	if !isAdmin {
		if user.GetDrawQuota() <= 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "今日画图次数已用完，请明天再来"})
			return
		}
	}

	// Select image channels
	channels, err := model.SelectChannels(modelName)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "没有可用渠道处理该模型"})
		return
	}

	// Build OpenAI image generation request body
	upstreamBody := map[string]interface{}{
		"model":  modelName,
		"prompt": req.Prompt,
		"n":      1,
	}
	if size != "" && size != "auto" {
		upstreamBody["size"] = size
	}
	if quality != "" {
		upstreamBody["quality"] = quality
	}
	bodyBytes, _ := json.Marshal(upstreamBody)

	// Forward to upstream
	client := &http.Client{Timeout: 120 * time.Second}
	var resp *http.Response
	var usedChannel *model.Channel

	for i := range channels {
		ch := &channels[i]
		upstreamURL := buildUpstreamURL(ch, "/v1/images/generations", "", false)
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
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游请求失败"})
		return
	}
	defer resp.Body.Close()

	// Read full response
	bodyBytes2, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "读取上游响应失败"})
		return
	}

	// Detect SSE and parse
	var imageData string
	contentType := resp.Header.Get("Content-Type")
	isSSE := strings.Contains(contentType, "text/event-stream")

	if !isSSE {
		// Peek for SSE prefix
		if strings.HasPrefix(string(bodyBytes2), "data: ") {
			isSSE = true
		}
	}

	if isSSE {
		// Parse SSE: find last completed or last data line
		var lastData string
		var lastCompleted string
		for _, line := range strings.Split(string(bodyBytes2), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "data: ") {
				continue
			}
			data := strings.TrimPrefix(trimmed, "data: ")
			if data == "[DONE]" || data == "" {
				continue
			}
			lastData = data
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
		imageData = lastData
	} else {
		imageData = string(bodyBytes2)
	}

	if imageData == "" {
		preview := string(bodyBytes2)
		if len(preview) > 500 {
			preview = preview[:500]
		}
		log.Printf("[draw] upstream returned empty image data: status=%d content-type=%s body=%s",
			resp.StatusCode, contentType, preview)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":         "上游返回空响应",
			"upstream_code": resp.StatusCode,
			"content_type":  contentType,
			"body_preview":  preview,
		})
		return
	}

	// Parse the image data and extract URLs
	var imageURLs []gin.H

	// Try grsai format {"results":[{"url":"..."}]}
	var grsai struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	}
	if json.Unmarshal([]byte(imageData), &grsai) == nil && len(grsai.Results) > 0 {
		for _, r := range grsai.Results {
			imageURLs = append(imageURLs, gin.H{"url": r.URL})
		}
	}

	// Try standard OpenAI image format {"data":[{"url":"..."}]}
	if imageURLs == nil {
		var oai struct {
			Data []struct {
				URL     string `json:"url"`
				B64JSON string `json:"b64_json"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(imageData), &oai) == nil && len(oai.Data) > 0 {
			for _, d := range oai.Data {
				item := gin.H{"url": d.URL}
				if d.B64JSON != "" {
					item["b64_json"] = d.B64JSON
				}
				imageURLs = append(imageURLs, item)
			}
		}
	}

	// Try generic JSON with data field
	if imageURLs == nil {
		var generic map[string]interface{}
		if json.Unmarshal([]byte(imageData), &generic) == nil {
			if data, ok := generic["data"]; ok && data != nil {
				if arr, ok := data.([]interface{}); ok {
					for _, item := range arr {
						if obj, ok := item.(map[string]interface{}); ok {
							entry := gin.H{}
							if u, ok := obj["url"].(string); ok {
								entry["url"] = u
							}
							if b, ok := obj["b64_json"].(string); ok {
								entry["b64_json"] = b
							}
							imageURLs = append(imageURLs, entry)
						}
					}
				}
			}
		}
	}

	// Fallback
	if imageURLs == nil {
		imageURLs = []gin.H{}
	}

	// Deduct quota (non-admin) and write a simple success log
	quotaRemaining := -1
	if !isAdmin {
		if user.DeductDrawQuota() {
			quotaRemaining = user.DailyDrawQuota
		}
	}

	// Write a minimal log entry (no billing)
	enqueueLog(logTask{
		user:        user,
		token:       &model.Token{Name: "draw-web"},
		channel:     usedChannel,
		modelName:   modelName,
		statusCode:  http.StatusOK,
		path:        "/draw",
		preDeducted: 0,
		startTime:   time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{
		"created":         time.Now().Unix(),
		"data":            imageURLs,
		"quota_remaining": quotaRemaining,
	})
}
