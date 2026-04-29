package handler

import (
	"new-api-lite/model"
	"time"
)

// logTask bundles all parameters needed for recordLog + optional threshold check.
type logTask struct {
	user            *model.User
	token           *model.Token
	channel         *model.Channel
	modelName       string
	inputTokens     int
	outputTokens    int
	cacheTokens     int
	statusCode      int
	path            string
	preDeducted     float64
	startTime       time.Time
	checkThresholds bool
}

const logWorkers = 8
const logQueueSize = 2048

var logQueue = make(chan logTask, logQueueSize)

func init() {
	for i := 0; i < logWorkers; i++ {
		go logWorker()
	}
}

func logWorker() {
	for task := range logQueue {
		recordLog(task.user, task.token, task.channel, task.modelName,
			task.inputTokens, task.outputTokens, task.cacheTokens,
			task.statusCode, task.path, task.preDeducted, task.startTime)
		if task.checkThresholds && task.user != nil {
			checkUsageThresholds(task.user)
		}
	}
}

// enqueueLog submits a logging task to the worker pool.
// Falls back to synchronous execution if the queue is full (overflow valve).
func enqueueLog(task logTask) {
	select {
	case logQueue <- task:
	default:
		// Queue full — execute inline to avoid dropping logs
		recordLog(task.user, task.token, task.channel, task.modelName,
			task.inputTokens, task.outputTokens, task.cacheTokens,
			task.statusCode, task.path, task.preDeducted, task.startTime)
		if task.checkThresholds && task.user != nil {
			checkUsageThresholds(task.user)
		}
	}
}
