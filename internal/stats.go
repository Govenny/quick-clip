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

// ContextStats stores slot pinnings and item usage records for a specific context.
type ContextStats struct {
	Pinned  [3]string              `json:"pinned"`
	Records map[string]UsageRecord `json:"records"`
}

// StatsFileV2 represents the V2 serialization schema of usage_stats.json.
type StatsFileV2 struct {
	Version  int                      `json:"version"`
	Contexts map[string]*ContextStats `json:"contexts"`
}

// CapsuleSlot represents an individual capsule slot state for the frontend.
type CapsuleSlot struct {
	Slot   int    `json:"slot"`   // 1, 2, 3
	Type   string `json:"type"`   // "pinned", "auto", "empty"
	ItemId string `json:"itemId"` // "" if empty
}

// StatsManager manages context-aware usage frequency statistics and pinned slots.
type StatsManager struct {
	mu        sync.RWMutex
	filePath  string
	contexts  map[string]*ContextStats
	saveMu    sync.Mutex
	saveTimer *time.Timer
}

// NewStatsManager initializes a StatsManager loading from the specified data directory.
func NewStatsManager(dataDir string) *StatsManager {
	os.MkdirAll(dataDir, 0755)
	filePath := filepath.Join(dataDir, "usage_stats.json")

	sm := &StatsManager{
		filePath: filePath,
		contexts: make(map[string]*ContextStats),
	}
	sm.load()
	return sm
}

func (sm *StatsManager) load() {
	raw, err := os.ReadFile(sm.filePath)
	if err != nil {
		return
	}

	// 1. 尝试解析 V2 格式
	var v2 StatsFileV2
	if err := json.Unmarshal(raw, &v2); err == nil && v2.Version >= 2 && v2.Contexts != nil {
		sm.contexts = v2.Contexts
		for _, ctx := range sm.contexts {
			if ctx != nil && ctx.Records == nil {
				ctx.Records = make(map[string]UsageRecord)
			}
		}
		return
	}

	// 2. 兼容降级：若失败或为旧文件，按 V1 格式 (map[string]map[string]UsageRecord) 解析
	var legacy map[string]map[string]UsageRecord
	if err := json.Unmarshal(raw, &legacy); err == nil {
		sm.contexts = make(map[string]*ContextStats)
		for cKey, records := range legacy {
			if records == nil {
				records = make(map[string]UsageRecord)
			}
			sm.contexts[cKey] = &ContextStats{
				Pinned:  [3]string{"", "", ""},
				Records: records,
			}
		}
		sm.scheduleSave()
		return
	}

	sm.contexts = make(map[string]*ContextStats)
}

// Flush synchronously writes the current stats to disk atomically in V2 format.
func (sm *StatsManager) Flush() error {
	sm.saveMu.Lock()
	defer sm.saveMu.Unlock()

	if sm.saveTimer != nil {
		sm.saveTimer.Stop()
		sm.saveTimer = nil
	}

	sm.mu.RLock()
	v2Data := StatsFileV2{
		Version:  2,
		Contexts: sm.contexts,
	}
	raw, err := json.MarshalIndent(v2Data, "", "  ")
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
	ctx, exists := sm.contexts[contextKey]
	if !exists || ctx == nil {
		ctx = &ContextStats{
			Pinned:  [3]string{"", "", ""},
			Records: make(map[string]UsageRecord),
		}
		sm.contexts[contextKey] = ctx
	}
	if ctx.Records == nil {
		ctx.Records = make(map[string]UsageRecord)
	}

	rec := ctx.Records[itemId]
	rec.Count++
	rec.LastUsed = time.Now().Unix()
	ctx.Records[itemId] = rec
	sm.mu.Unlock()

	sm.scheduleSave()
}

// PinSlot fixes an itemId into a specified slot (1-based: 1, 2, 3) for the context.
func (sm *StatsManager) PinSlot(contextKey string, slot int, itemId string) {
	if contextKey == "" || slot < 1 || slot > 3 {
		return
	}
	idx := slot - 1

	sm.mu.Lock()
	ctx, exists := sm.contexts[contextKey]
	if !exists || ctx == nil {
		ctx = &ContextStats{
			Pinned:  [3]string{"", "", ""},
			Records: make(map[string]UsageRecord),
		}
		sm.contexts[contextKey] = ctx
	}
	ctx.Pinned[idx] = itemId
	sm.mu.Unlock()

	sm.scheduleSave()
}

// UnpinSlot releases a specified slot (1-based: 1, 2, 3) for the context back to auto.
func (sm *StatsManager) UnpinSlot(contextKey string, slot int) {
	if contextKey == "" || slot < 1 || slot > 3 {
		return
	}
	idx := slot - 1

	sm.mu.Lock()
	if ctx, exists := sm.contexts[contextKey]; exists && ctx != nil {
		ctx.Pinned[idx] = ""
	}
	sm.mu.Unlock()

	sm.scheduleSave()
}

// RemoveItemStats removes all usage records and pinned slot references for the specified itemId across all contexts.
func (sm *StatsManager) RemoveItemStats(itemId string) {
	if itemId == "" {
		return
	}

	sm.mu.Lock()
	for _, ctx := range sm.contexts {
		if ctx == nil {
			continue
		}
		delete(ctx.Records, itemId)
		for i := 0; i < 3; i++ {
			if ctx.Pinned[i] == itemId {
				ctx.Pinned[i] = ""
			}
		}
	}
	sm.mu.Unlock()

	sm.scheduleSave()
}

func (sm *StatsManager) getEffectivePinned(contextKey string) [3]string {
	// 1. 完全精确匹配当前场景
	if ctx, exists := sm.contexts[contextKey]; exists && ctx != nil {
		if ctx.Pinned[0] != "" || ctx.Pinned[1] != "" || ctx.Pinned[2] != "" {
			return ctx.Pinned
		}
	}

	// 2. 浏览器网页场景回退继承主进程通用槽位
	parts := strings.SplitN(contextKey, "::", 2)
	procName := parts[0]
	if len(parts) > 1 && procName != "" {
		if procCtx, exists := sm.contexts[procName]; exists && procCtx != nil {
			return procCtx.Pinned
		}
	}

	return [3]string{"", "", ""}
}

// GetContextSlots returns exactly 3 slots for the given contextKey combining pinned slots and auto usage recommendations.
func (sm *StatsManager) GetContextSlots(contextKey string) []CapsuleSlot {
	slots := make([]CapsuleSlot, 3)
	for i := 0; i < 3; i++ {
		slots[i] = CapsuleSlot{
			Slot:   i + 1,
			Type:   "empty",
			ItemId: "",
		}
	}

	if contextKey == "" {
		return slots
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 1. 获取有效固定槽位
	pinned := sm.getEffectivePinned(contextKey)
	pinnedSet := make(map[string]bool)
	for i := 0; i < 3; i++ {
		if pinned[i] != "" {
			slots[i].Type = "pinned"
			slots[i].ItemId = pinned[i]
			pinnedSet[pinned[i]] = true
		}
	}

	// 2. 获取使用频次最高项，严格剔除已锁定的条目
	autoCandidates := sm.getTopItemIdsLocked(contextKey, 3, pinnedSet)

	// 3. 填入未被锁定的空闲槽位
	autoIdx := 0
	for i := 0; i < 3; i++ {
		if slots[i].Type != "pinned" {
			if autoIdx < len(autoCandidates) {
				slots[i].Type = "auto"
				slots[i].ItemId = autoCandidates[autoIdx]
				autoIdx++
			} else {
				slots[i].Type = "empty"
				slots[i].ItemId = ""
			}
		}
	}

	return slots
}

// GetTopItemIds returns the top item IDs for backward compatibility.
func (sm *StatsManager) GetTopItemIds(contextKey string, limit int) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.getTopItemIdsLocked(contextKey, limit, nil)
}

type itemScore struct {
	itemId   string
	count    int
	lastUsed int64
}

func (sm *StatsManager) getTopItemIdsLocked(contextKey string, limit int, excludeIds map[string]bool) []string {
	if contextKey == "" || limit <= 0 {
		return nil
	}

	// 1. 完全精确匹配
	if ctx, exists := sm.contexts[contextKey]; exists && ctx != nil && len(ctx.Records) > 0 {
		return extractTopScores(ctx.Records, limit, excludeIds)
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
		for k, ctx := range sm.contexts {
			if ctx == nil || !strings.HasPrefix(k, procName+"::") {
				continue
			}
			storedTitle := strings.ToLower(strings.TrimPrefix(k, procName+"::"))
			if strings.Contains(queryTitle, storedTitle) || strings.Contains(storedTitle, queryTitle) {
				for id, rec := range ctx.Records {
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
			return extractTopScores(fuzzyMerged, limit, excludeIds)
		}
	}

	// 3. 进程级通用兜底（同应用下使用频次最高项）
	procMerged := make(map[string]UsageRecord)
	for k, ctx := range sm.contexts {
		if ctx == nil {
			continue
		}
		if k == procName || strings.HasPrefix(k, procName+"::") {
			for id, rec := range ctx.Records {
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
		return extractTopScores(procMerged, limit, excludeIds)
	}

	return nil
}

func extractTopScores(records map[string]UsageRecord, limit int, excludeIds map[string]bool) []string {
	scores := make([]itemScore, 0, len(records))
	for id, rec := range records {
		if excludeIds != nil && excludeIds[id] {
			continue
		}
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
