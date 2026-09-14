package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestStatsManager_RecordAndGetTop(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewStatsManager(tempDir)

	contextKey := "navicat.exe"

	// 记录 item1 2次，item2 5次，item3 1次
	sm.RecordUsage(contextKey, "item1")
	sm.RecordUsage(contextKey, "item1")

	for i := 0; i < 5; i++ {
		sm.RecordUsage(contextKey, "item2")
	}

	sm.RecordUsage(contextKey, "item3")

	top := sm.GetTopItemIds(contextKey, 2)
	wantTop := []string{"item2", "item1"}
	if !reflect.DeepEqual(top, wantTop) {
		t.Fatalf("GetTopItemIds got %v, want %v", top, wantTop)
	}

	// 显式刷新落盘，验证持久化与重新加载
	if err := sm.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	smReloaded := NewStatsManager(tempDir)
	topReloaded := smReloaded.GetTopItemIds(contextKey, 3)
	wantTop3 := []string{"item2", "item1", "item3"}
	if !reflect.DeepEqual(topReloaded, wantTop3) {
		t.Fatalf("Reloaded GetTopItemIds got %v, want %v", topReloaded, wantTop3)
	}

	// 测试从统计中删除 item2
	smReloaded.RemoveItemStats("item2")
	if err := smReloaded.Flush(); err != nil {
		t.Fatalf("Flush after RemoveItemStats failed: %v", err)
	}
	topAfterRemove := smReloaded.GetTopItemIds(contextKey, 3)
	wantAfterRemove := []string{"item1", "item3"}
	if !reflect.DeepEqual(topAfterRemove, wantAfterRemove) {
		t.Fatalf("GetTopItemIds after RemoveItemStats got %v, want %v", topAfterRemove, wantAfterRemove)
	}
}

func TestGenerateContextKey(t *testing.T) {
	tests := []struct {
		procName string
		title    string
		want     string
	}{
		{
			procName: "chrome.exe",
			title:    "(3) 登录 - GitHub - Google Chrome",
			want:     "chrome.exe::登录 - GitHub",
		},
		{
			procName: "msedge.exe",
			title:    "(99+) PROJ-1234缺陷详情 - Microsoft Edge",
			want:     "msedge.exe::PROJ-1234缺陷详情",
		},
		{
			procName: "chrome.exe",
			title:    "[生产] 订单管理系统 - Google Chrome",
			want:     "chrome.exe::[生产] 订单管理系统",
		},
		{
			procName: "msedge.exe",
			title:    "视觉算法管理平台 和另外 11 个页面 - 用户1 - Microsoft Edge",
			want:     "msedge.exe::视觉算法管理平台",
		},
		{
			procName: "navicat.exe",
			title:    "MySQL connection - localhost",
			want:     "navicat.exe",
		},
		{
			procName: "Windowsterminal.exe",
			title:    "PowerShell - Administrator",
			want:     "windowsterminal.exe",
		},
	}

	for _, tt := range tests {
		got := GenerateContextKey(tt.procName, tt.title)
		if got != tt.want {
			t.Errorf("GenerateContextKey(%q, %q) = %q, want %q", tt.procName, tt.title, got, tt.want)
		}
	}
}

func TestStatsManager_FuzzyMatch(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewStatsManager(tempDir)

	// 模拟用户库中存的旧记录（带有残余噪音）
	oldKey := "msedge.exe::视觉算法管理平台 和另外 11 个页面 - 用户1 - Microsoft E"
	sm.RecordUsage(oldKey, "target-password-uuid")

	// 此时清洗后的干净场景指纹发起查询
	cleanKey := "msedge.exe::视觉算法管理平台"
	top := sm.GetTopItemIds(cleanKey, 1)

	if len(top) == 0 || top[0] != "target-password-uuid" {
		t.Fatalf("Fuzzy match failed, got %v, want ['target-password-uuid']", top)
	}

	// 另外测试：库中存的是干净的，查询带着新网页副标题
	sm.RecordUsage("chrome.exe::GitHub", "github-token-uuid")
	topGithub := sm.GetTopItemIds("chrome.exe::GitHub: Where the world builds software", 1)
	if len(topGithub) == 0 || topGithub[0] != "github-token-uuid" {
		t.Fatalf("Fuzzy match for GitHub failed, got %v", topGithub)
	}
}

func TestStatsManager_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewStatsManager(tempDir)

	var wg sync.WaitGroup
	const workers = 20
	const iterations = 50

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				cKey := fmt.Sprintf("proc%d.exe", workerId%4)
				itemId := fmt.Sprintf("item-%d", i%5)
				sm.RecordUsage(cKey, itemId)
				_ = sm.GetTopItemIds(cKey, 3)
			}
		}(w)
	}

	wg.Wait()

	if err := sm.Flush(); err != nil {
		t.Fatalf("Flush failed after concurrent operations: %v", err)
	}

	// Verify reload
	smReloaded := NewStatsManager(tempDir)
	top := smReloaded.GetTopItemIds("proc0.exe", 3)
	if len(top) == 0 {
		t.Fatalf("Expected non-empty top items for proc0.exe after concurrency test")
	}
}

func TestStatsManager_LegacyV1Migration(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "usage_stats.json")

	// 写入合法的 V1 旧版格式文件（无 version, 根为 map[contextKey]map[itemId]UsageRecord）
	legacyJSON := `{
		"code.exe": {
			"item-legacy-1": { "count": 10, "lastUsed": 1600000000 },
			"item-legacy-2": { "count": 5, "lastUsed": 1600000000 }
		}
	}`
	if err := os.WriteFile(filePath, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("Failed to write legacy file: %v", err)
	}

	sm := NewStatsManager(tempDir)
	slots := sm.GetContextSlots("code.exe")

	if len(slots) != 3 {
		t.Fatalf("Expected 3 slots, got %d", len(slots))
	}

	// 槽位 1 应当自动填充最高频的 item-legacy-1
	if slots[0].Slot != 1 || slots[0].Type != "auto" || slots[0].ItemId != "item-legacy-1" {
		t.Errorf("Slot 1 unexpected: %+v", slots[0])
	}
	// 槽位 2 应当自动填充次高频的 item-legacy-2
	if slots[1].Slot != 2 || slots[1].Type != "auto" || slots[1].ItemId != "item-legacy-2" {
		t.Errorf("Slot 2 unexpected: %+v", slots[1])
	}
	// 槽位 3 应当为空槽 empty
	if slots[2].Slot != 3 || slots[2].Type != "empty" || slots[2].ItemId != "" {
		t.Errorf("Slot 3 unexpected: %+v", slots[2])
	}

	// 验证落盘后自动转为 V2 格式
	if err := sm.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Read upgraded file failed: %v", err)
	}
	if !strings.Contains(string(raw), `"version": 2`) {
		t.Errorf("Expected upgraded JSON to contain version 2, got: %s", string(raw))
	}
}

func TestStatsManager_PinSlotAndExclusion(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewStatsManager(tempDir)
	cKey := "notepad.exe"

	// 产生使用记录：itemA: 10次, itemB: 8次, itemC: 5次, itemD: 2次
	for i := 0; i < 10; i++ {
		sm.RecordUsage(cKey, "itemA")
	}
	for i := 0; i < 8; i++ {
		sm.RecordUsage(cKey, "itemB")
	}
	for i := 0; i < 5; i++ {
		sm.RecordUsage(cKey, "itemC")
	}
	for i := 0; i < 2; i++ {
		sm.RecordUsage(cKey, "itemD")
	}

	// 初始状态：槽位 1 -> itemA (auto), 槽位 2 -> itemB (auto), 槽位 3 -> itemC (auto)
	initialSlots := sm.GetContextSlots(cKey)
	if initialSlots[0].ItemId != "itemA" || initialSlots[1].ItemId != "itemB" || initialSlots[2].ItemId != "itemC" {
		t.Fatalf("Initial slots unexpected: %+v", initialSlots)
	}

	// 手动固定 槽位 2 为 itemD（即使 itemD 次数很少）
	sm.PinSlot(cKey, 2, "itemD")

	// 期望：
	// 槽位 1 (未锁定) -> itemA (auto, 频次第1)
	// 槽位 2 (锁定)   -> itemD (pinned)
	// 槽位 3 (未锁定) -> itemB (auto, 频次第2)
	slots := sm.GetContextSlots(cKey)
	if slots[0].Slot != 1 || slots[0].Type != "auto" || slots[0].ItemId != "itemA" {
		t.Errorf("Slot 1 unexpected after pin: %+v", slots[0])
	}
	if slots[1].Slot != 2 || slots[1].Type != "pinned" || slots[1].ItemId != "itemD" {
		t.Errorf("Slot 2 unexpected after pin: %+v", slots[1])
	}
	if slots[2].Slot != 3 || slots[2].Type != "auto" || slots[2].ItemId != "itemB" {
		t.Errorf("Slot 3 unexpected after pin: %+v", slots[2])
	}

	// 如果手动固定 槽位 1 为 itemA：
	// 此时 itemA 已被固定，不应再重复占用槽位 3！
	sm.PinSlot(cKey, 1, "itemA")
	slots2 := sm.GetContextSlots(cKey)
	// 槽位 1 -> itemA (pinned)
	// 槽位 2 -> itemD (pinned)
	// 槽位 3 -> itemB (auto)
	if slots2[0].Type != "pinned" || slots2[0].ItemId != "itemA" {
		t.Errorf("Slot 1 unexpected: %+v", slots2[0])
	}
	if slots2[1].Type != "pinned" || slots2[1].ItemId != "itemD" {
		t.Errorf("Slot 2 unexpected: %+v", slots2[1])
	}
	if slots2[2].Type != "auto" || slots2[2].ItemId != "itemB" {
		t.Errorf("Slot 3 unexpected: %+v", slots2[2])
	}

	// 取消固定 槽位 2
	sm.UnpinSlot(cKey, 2)
	slots3 := sm.GetContextSlots(cKey)
	// 槽位 1 -> itemA (pinned)
	// 槽位 2 -> itemB (auto, 因为 itemA 已被 pinned 排除)
	// 槽位 3 -> itemC (auto)
	if slots3[0].Type != "pinned" || slots3[0].ItemId != "itemA" {
		t.Errorf("Slot 1 unexpected: %+v", slots3[0])
	}
	if slots3[1].Type != "auto" || slots3[1].ItemId != "itemB" {
		t.Errorf("Slot 2 unexpected: %+v", slots3[1])
	}
	if slots3[2].Type != "auto" || slots3[2].ItemId != "itemC" {
		t.Errorf("Slot 3 unexpected: %+v", slots3[2])
	}
}

func TestStatsManager_BrowserFallbackAndRemoveItem(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewStatsManager(tempDir)

	// 在主进程 chrome.exe 固定槽位 1
	sm.PinSlot("chrome.exe", 1, "pinned-for-chrome")

	// 查询子网页场景
	slots := sm.GetContextSlots("chrome.exe::GitHub - Pull Requests")
	if slots[0].Type != "pinned" || slots[0].ItemId != "pinned-for-chrome" {
		t.Errorf("Browser fallback failed, slot 0 = %+v", slots[0])
	}

	// 删除条目时，对应的固定引用应一并被清除
	sm.RemoveItemStats("pinned-for-chrome")
	slotsAfterRemove := sm.GetContextSlots("chrome.exe::GitHub - Pull Requests")
	if slotsAfterRemove[0].ItemId == "pinned-for-chrome" {
		t.Errorf("Expected pinned reference to be cleared after RemoveItemStats, got %+v", slotsAfterRemove[0])
	}
}

