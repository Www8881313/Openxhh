package ai

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"openxhh/loger"

	"go.uber.org/zap"
)

const tokenRecordFileName = "token_records.jsonl"

var tokenRecordMu sync.Mutex

type tokenRecord struct {
	Time   string `json:"time"`
	Model  string `json:"model,omitempty"`
	Tokens int    `json:"tokens"`
}

func appendTokenRecord(model string, tokens int) {
	if tokens <= 0 {
		return
	}
	tokenRecordMu.Lock()
	defer tokenRecordMu.Unlock()

	file, err := os.OpenFile(tokenRecordFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		loger.Loger.Warn("[Ai]无法写入token记录", zap.Error(err))
		return
	}
	defer file.Close()

	record := tokenRecord{
		Time:   time.Now().Format("2006-01-02 15:04:05"),
		Model:  model,
		Tokens: tokens,
	}
	if err := json.NewEncoder(file).Encode(record); err != nil {
		loger.Loger.Warn("[Ai]无法写入token记录", zap.Error(err))
	}
}
