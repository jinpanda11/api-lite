package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"new-api-lite/model"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ModelStatus holds the latest connectivity test result for a specific model.
type ModelStatus struct {
	Model       string `json:"model"`
	ChannelName string `json:"channel_name"`
	ChannelID   uint   `json:"channel_id"`
	Online      bool   `json:"online"`
	StatusCode  int    `json:"status_code"`
	LatencyMs   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
	CheckedAt   int64  `json:"checked_at"`
}

var (
	statusMu    sync.RWMutex
	statusCache = map[string]*ModelStatus{} // key: "model|channelID"

	monitorInterval = 5 * time.Minute
	monitorTicker   *time.Ticker
	monitorStop     = make(chan struct{})
)

func cacheKey(model string, channelID uint) string {
	return model + "|" + strconv.Itoa(int(channelID))
}

// StartMonitor begins the periodic model connectivity tester.
func StartMonitor() {
	go func() {
		loadMonitorInterval()
		runAllTests()

		monitorTicker = time.NewTicker(monitorInterval)
		defer monitorTicker.Stop()

		for {
			select {
			case <-monitorTicker.C:
				loadMonitorInterval()
				runAllTests()
			case <-monitorStop:
				return
			}
		}
	}()
}

func loadMonitorInterval() {
	var s model.Setting
	if err := model.DB.Where("key = ?", "monitor_interval").First(&s).Error; err == nil {
		if mins, err := strconv.Atoi(s.Value); err == nil && mins > 0 {
			monitorInterval = time.Duration(mins) * time.Minute
		}
	}
}

func applyMonitorInterval(minutes int) {
	monitorInterval = time.Duration(minutes) * time.Minute
	model.DB.Save(&model.Setting{Key: "monitor_interval", Value: strconv.Itoa(minutes)})
	if monitorTicker != nil {
		monitorTicker.Reset(monitorInterval)
	}
}

func runAllTests() {
	var channels []model.Channel
	model.DB.Where("status = 1 AND monitor_enabled = 1").Find(&channels)

	// Collect all model+channel pairs
	type testItem struct {
		modelName string
		channel   model.Channel
	}
	var items []testItem
	for _, ch := range channels {
		if ch.Models == "" {
			// Channel has no explicit models, skip (auto-discover not practical for per-model test)
			continue
		}
		for _, m := range splitModels(ch.Models) {
			if m == "" {
				continue
			}
			items = append(items, testItem{modelName: m, channel: ch})
		}
	}

	var wg sync.WaitGroup
	for _, it := range items {
		wg.Add(1)
		go func(m string, ch model.Channel) {
			defer wg.Done()
			result := testModelConnectivity(m, ch)
			statusMu.Lock()
			statusCache[cacheKey(m, ch.ID)] = result
			statusMu.Unlock()
		}(it.modelName, it.channel)
	}
	wg.Wait()
}

func testModelConnectivity(modelName string, ch model.Channel) *ModelStatus {
	result := &ModelStatus{
		Model:       modelName,
		ChannelName: ch.Name,
		ChannelID:   ch.ID,
	}

	baseURL := strings.TrimRight(ch.BaseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	client := &http.Client{Timeout: 30 * time.Second}

	var testURL, method string
	var body string

	// Build request body with proper JSON encoding (prevents model name injection)
	reqBody := map[string]interface{}{
		"model":    modelName,
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"max_tokens": 5,
		"stream":   false,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	chatBody := string(bodyBytes)

	imageReqBody := map[string]interface{}{
		"model":  modelName,
		"prompt": "test",
		"n":      1,
		"size":   "256x256",
	}
	imageBodyBytes, _ := json.Marshal(imageReqBody)
	imageBody := string(imageBodyBytes)

	if ch.Type == "image" {
		if ch.FixedPath != "" {
			testURL = baseURL + ch.FixedPath
		} else {
			testURL = baseURL + "/v1/images/generations"
		}
		method = "POST"
		body = imageBody
	} else {
		testURL = baseURL + "/v1/chat/completions"
		method = "POST"
		body = chatBody
	}

	req, err := http.NewRequest(method, testURL, strings.NewReader(body))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Authorization", "Bearer "+ch.APIKey)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMs = time.Since(start).Milliseconds()
	result.CheckedAt = time.Now().Unix()

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode

	// Read response body to check for actual content (not just HTTP status)
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var respBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	// Online = 2xx AND response body has actual content (choices with text)
	result.Online = false
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if json.Unmarshal(respBytes, &respBody) == nil {
			if len(respBody.Choices) > 0 && (respBody.Choices[0].Message.Content != "" || respBody.Choices[0].Delta.Content != "") {
				result.Online = true
			} else if respBody.Error != nil && respBody.Error.Message != "" {
				result.Error = respBody.Error.Message
			}
		} else {
			// Non-JSON body, if 2xx treat as online
			result.Online = true
		}
	} else if respBody.Error != nil && respBody.Error.Message != "" {
		result.Error = respBody.Error.Message
	}

	return result
}

// RefreshStatus runs all tests immediately, then returns the updated status.
// POST /api/status
func RefreshStatus(c *gin.Context) {
	runAllTests()
	GetStatus(c)
}

// GetStatus returns connectivity status for all monitored models.
// GET /api/status
func GetStatus(c *gin.Context) {
	statusMu.RLock()
	defer statusMu.RUnlock()

	var list []*ModelStatus
	for _, s := range statusCache {
		list = append(list, s)
	}

	if list == nil {
		list = []*ModelStatus{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     list,
		"interval": int(monitorInterval.Minutes()),
	})
}

// ToggleChannelMonitor enables or disables monitoring for a channel.
// PUT /api/admin/channel/:id/monitor  { monitor_enabled: true/false }
func ToggleChannelMonitor(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		MonitorEnabled bool `json:"monitor_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ch model.Channel
	if err := model.DB.First(&ch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	model.DB.Model(&ch).Update("monitor_enabled", req.MonitorEnabled)
	audit(c, "toggle_monitor", fmt.Sprintf("channel=%s enabled=%v", ch.Name, req.MonitorEnabled))

	if !req.MonitorEnabled {
		// Remove all model results for this channel
		statusMu.Lock()
		for _, m := range splitModels(ch.Models) {
			delete(statusCache, cacheKey(m, ch.ID))
		}
		statusMu.Unlock()
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("monitor %s: %v", ch.Name, req.MonitorEnabled)})
}

// GetMonitorConfig returns monitor interval and per-channel enabled status.
// GET /api/admin/monitor-config
func GetMonitorConfig(c *gin.Context) {
	var channels []model.Channel
	model.DB.Select("id, name, monitor_enabled").Order("id").Find(&channels)

	interval := int(monitorInterval.Minutes())

	c.JSON(http.StatusOK, gin.H{
		"interval": interval,
		"channels": channels,
	})
}

// UpdateMonitorConfig sets the monitor interval.
// PUT /api/admin/monitor-config  { interval: 5 }
func UpdateMonitorConfig(c *gin.Context) {
	var req struct {
		Interval int `json:"interval" binding:"required,min=1,max=1440"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applyMonitorInterval(req.Interval)
	audit(c, "update_monitor_interval", fmt.Sprintf("minutes=%d", req.Interval))
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("interval set to %d minutes", req.Interval)})
}
