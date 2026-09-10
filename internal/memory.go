package internal

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modpsapi             = syscall.NewLazyDLL("psapi.dll")
	procEmptyWorkingSet  = modpsapi.NewProc("EmptyWorkingSet")
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32 = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First   = modkernel32.NewProc("Process32FirstW")
	procProcess32Next    = modkernel32.NewProc("Process32NextW")
	procOpenProcess      = modkernel32.NewProc("OpenProcess")
	procCloseHandle      = modkernel32.NewProc("CloseHandle")
)

const (
	th32csSnapProcess       = 0x00000002
	processQueryInformation = 0x0400
	processSetQuota         = 0x0100
)

type processEntry32W struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

// MemoryTrimmer manages delayed memory trimming on idle/hide.
type MemoryTrimmer struct {
	mu    sync.Mutex
	timer *time.Timer
}

func NewMemoryTrimmer() *MemoryTrimmer {
	return &MemoryTrimmer{}
}

// ScheduleTrim schedules memory trimming after the specified delay.
func (m *MemoryTrimmer) ScheduleTrim(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.timer != nil {
		m.timer.Stop()
	}
	m.timer = time.AfterFunc(delay, func() {
		TrimProcessFamily()
	})
}

// CancelTrim cancels any pending scheduled trimming.
func (m *MemoryTrimmer) CancelTrim() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// TrimProcessFamily releases unused memory for the main Go process and all child processes (WebView2).
func TrimProcessFamily() {
	// 1. 温和触发 Go GC，不调用阻塞系统调度的 debug.FreeOSMemory()
	runtime.GC()

	// 2. Trim main process working set
	if hCurrent, err := syscall.GetCurrentProcess(); err == nil {
		procEmptyWorkingSet.Call(uintptr(hCurrent))
	}

	// 3. Collect and trim all child/grandchild processes (msedgewebview2.exe tree)
	myPid := uint32(syscall.Getpid())
	childPids := getDescendantPIDs(myPid)
	for _, pid := range childPids {
		trimSingleProcess(pid)
	}
}

func trimSingleProcess(pid uint32) {
	hProc, _, _ := procOpenProcess.Call(
		uintptr(processQueryInformation|processSetQuota),
		0,
		uintptr(pid),
	)
	if hProc == 0 {
		return
	}
	defer procCloseHandle.Call(hProc)

	procEmptyWorkingSet.Call(hProc)
}

// getDescendantPIDs finds all descendants of the specified PID using Toolhelp snapshot.
func getDescendantPIDs(rootPid uint32) []uint32 {
	hSnap, _, _ := procCreateToolhelp32.Call(th32csSnapProcess, 0)
	if hSnap == uintptr(syscall.InvalidHandle) {
		return nil
	}
	defer procCloseHandle.Call(hSnap)

	var allParents = make(map[uint32][]uint32)
	var entry processEntry32W
	entry.dwSize = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		allParents[entry.th32ParentProcessID] = append(allParents[entry.th32ParentProcessID], entry.th32ProcessID)
		ret, _, _ = procProcess32Next.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	}

	// BFS traversal to gather all descendants
	var result []uint32
	var queue = []uint32{rootPid}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if children, exists := allParents[curr]; exists {
			for _, child := range children {
				result = append(result, child)
				queue = append(queue, child)
			}
		}
	}
	return result
}
