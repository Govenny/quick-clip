package internal

import (
	"fmt"
	"reflect"
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
