package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// UsageRecord stores the usage count and latest timestamp for an item in a specific context.
type UsageRecord struct {
	Count    int   `json:"count"`
	LastUsed int64 `json:"lastUsed"`
}

// StatsManager manages context-aware usage frequency statistics.
type StatsManager struct {
	mu       sync.RWMutex
	filePath string
	// map[contextKey]map[itemId]UsageRecord
	data      map[string]map[string]UsageRecord
	saveMu    sync.Mutex
	saveTimer *time.Timer
}

// NewStatsManager initializes a StatsManager loading from the specified data directory.
func NewStatsManager(dataDir string) *StatsManager {
	os.MkdirAll(dataDir, 0755)
	filePath := filepath.Join(dataDir, "usage_stats.json")

	sm := &StatsManager{
		filePath: filePath,
		data:     make(map[string]map[string]UsageRecord),
	}
	sm.load()
	return sm
}

func (sm *StatsManager) load() {
	raw, err := os.ReadFile(sm.filePath)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &sm.data)
	if sm.data == nil {
		sm.data = make(map[string]map[string]UsageRecord)
	}
}

// Flush synchronously writes the current stats to disk atomically.
func (sm *StatsManager) Flush() error {
	sm.saveMu.Lock()
	defer sm.saveMu.Unlock()

	if sm.saveTimer != nil {
		sm.saveTimer.Stop()
		sm.saveTimer = nil
	}

	sm.mu.RLock()
	raw, err := json.MarshalIndent(sm.data, "", "  ")
	sm.mu.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(sm.filePath)
	tmpFile, err := os.CreateTemp(dir, ".stats-*.tmp")
	if err != nil {
		return os.WriteFile(sm.filePath, raw, 0644)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(raw); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	_ = os.Remove(sm.filePath)
	if err := os.Rename(tmpPath, sm.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return os.WriteFile(sm.filePath, raw, 0644)
	}
	return nil
}

func (sm *StatsManager) scheduleSave() {
	sm.saveMu.Lock()
	defer sm.saveMu.Unlock()

	if sm.saveTimer != nil {
		sm.saveTimer.Stop()
	}
	sm.saveTimer = time.AfterFunc(300*time.Millisecond, func() {
		_ = sm.Flush()
	})
}

// RecordUsage records an item access under the given contextKey.
func (sm *StatsManager) RecordUsage(contextKey, itemId string) {
	if contextKey == "" || itemId == "" {
		return
	}

	sm.mu.Lock()
	if sm.data[contextKey] == nil {
		sm.data[contextKey] = make(map[string]UsageRecord)
	}

	rec := sm.data[contextKey][itemId]
	rec.Count++
	rec.LastUsed = time.Now().Unix()
	sm.data[contextKey][itemId] = rec
	sm.mu.Unlock()

	// Debounced atomic save to prevent concurrent write collisions
	sm.scheduleSave()
}

// RemoveItemStats removes all usage records for the specified itemId across all contexts.
func (sm *StatsManager) RemoveItemStats(itemId string) {
	if itemId == "" {
		return
	}

	sm.mu.Lock()
	for cKey, records := range sm.data {
		delete(records, itemId)
		if len(records) == 0 {
			delete(sm.data, cKey)
		}
	}
	sm.mu.Unlock()

	sm.scheduleSave()
}

type itemScore struct {
	itemId   string
	count    int
	lastUsed int64
}

// GetTopItemIds returns the top item IDs for the given contextKey, sorted by count and recency.
// Supports 3-tier matching:
// 1. Exact match on contextKey
// 2. Fuzzy substring match on the title within the same process
// 3. Process-level general fallback
func (sm *StatsManager) GetTopItemIds(contextKey string, limit int) []string {
	if contextKey == "" || limit <= 0 {
		return nil
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 1. 完全精确匹配
	if records, exists := sm.data[contextKey]; exists && len(records) > 0 {
		return extractTopScores(records, limit)
	}

	// 拆分 procName 与 queryTitle
	parts := strings.SplitN(contextKey, "::", 2)
	procName := parts[0]
	queryTitle := ""
	if len(parts) > 1 {
		queryTitle = strings.ToLower(strings.TrimSpace(parts[1]))
	}

	// 2. 模糊子串匹配（同进程下，标题互为包含关系）
	if queryTitle != "" {
		fuzzyMerged := make(map[string]UsageRecord)
		for k, records := range sm.data {
			if !strings.HasPrefix(k, procName+"::") {
				continue
			}
			storedTitle := strings.ToLower(strings.TrimPrefix(k, procName+"::"))
			if strings.Contains(queryTitle, storedTitle) || strings.Contains(storedTitle, queryTitle) {
				for id, rec := range records {
					curr := fuzzyMerged[id]
					curr.Count += rec.Count
					if rec.LastUsed > curr.LastUsed {
						curr.LastUsed = rec.LastUsed
					}
					fuzzyMerged[id] = curr
				}
			}
		}
		if len(fuzzyMerged) > 0 {
			return extractTopScores(fuzzyMerged, limit)
		}
	}

	// 3. 进程级通用兜底（同应用下使用频次最高项）
	procMerged := make(map[string]UsageRecord)
	for k, records := range sm.data {
		if k == procName || strings.HasPrefix(k, procName+"::") {
			for id, rec := range records {
				curr := procMerged[id]
				curr.Count += rec.Count
				if rec.LastUsed > curr.LastUsed {
					curr.LastUsed = rec.LastUsed
				}
				procMerged[id] = curr
			}
		}
	}
	if len(procMerged) > 0 {
		return extractTopScores(procMerged, limit)
	}

	return nil
}

func extractTopScores(records map[string]UsageRecord, limit int) []string {
	scores := make([]itemScore, 0, len(records))
	for id, rec := range records {
		scores = append(scores, itemScore{
			itemId:   id,
			count:    rec.Count,
			lastUsed: rec.LastUsed,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].count != scores[j].count {
			return scores[i].count > scores[j].count
		}
		return scores[i].lastUsed > scores[j].lastUsed
	})

	resultCount := limit
	if len(scores) < resultCount {
		resultCount = len(scores)
	}

	result := make([]string, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = scores[i].itemId
	}
	return result
}
