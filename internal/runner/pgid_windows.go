//go:build windows

// Windows process-group management using Job Objects.
//
// On Unix, the kernel's process group (PGID) mechanism allows terminating all
// processes in a group with a single signal. Windows has no equivalent, but
// Job Objects (CreateJobObject / AssignProcessToJobObject / TerminateJobObject)
// provide process-tree lifecycle control.
//
// Design:
//   - applyProcessGroup: sets CREATE_NEW_PROCESS_GROUP on the child process (existing).
//   - After the process starts, RegisterWindowsJobObject creates a Job Object,
//     assigns the process to it, and stores the handle keyed by PID.
//   - TerminateProcessGroup / KillProcessGroup (in stop_windows.go) look up the
//     Job Object and call TerminateJobObject to kill the entire process tree.
//   - isProcessGroupAlive (in wait_windows.go) queries the job's active process count.
//   - CleanupWindowsJobObject releases Job Object handles and removes map entries.
//
// See ISSUE-003 in docs/dev/issues.md for background.

package runner

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsJobRegistry maps PID → Job Object handle.
var (
	windowsJobMu       sync.RWMutex
	windowsJobRegistry = map[int]windows.Handle{}
)

func applyProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

// getProcessGroupID returns the PID itself as a PGID placeholder.
// The real grouping is managed via Job Objects; see RegisterWindowsJobObject.
func getProcessGroupID(pid int) (int, error) {
	return pid, nil
}

// RegisterWindowsJobObject creates a Windows Job Object, assigns the process
// with the given PID to it, and stores the handle for later use by
// TerminateProcessGroup and isProcessGroupAlive.
//
// Call this after the process has started (i.e., after cmd.Start() returns).
// Returns nil if Job Objects are unavailable or unsupported (non-fatal fallback
// to single-PID kill).
func RegisterWindowsJobObject(pid int) error {
	if pid <= 0 {
		return nil
	}

	// Open the process with the permissions needed to assign it to a job.
	procHandle, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return nil // Non-fatal: fall back to PID-only termination.
	}

	// Create a Job Object.
	jobHandle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		windows.CloseHandle(procHandle) //nolint:errcheck
		return nil
	}

	// Configure kill-on-close: when the last handle to the job is closed,
	// all processes in the job are terminated automatically.
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, _ = windows.SetInformationJobObject(
		jobHandle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)

	// Assign the process to the job.
	if err := windows.AssignProcessToJobObject(jobHandle, procHandle); err != nil {
		windows.CloseHandle(jobHandle)  //nolint:errcheck
		windows.CloseHandle(procHandle) //nolint:errcheck
		return nil
	}
	windows.CloseHandle(procHandle) //nolint:errcheck

	windowsJobMu.Lock()
	windowsJobRegistry[pid] = jobHandle
	windowsJobMu.Unlock()
	return nil
}

// CleanupWindowsJobObject releases the Job Object handle for the given PID
// and removes it from the registry. Safe to call multiple times.
func CleanupWindowsJobObject(pid int) {
	windowsJobMu.Lock()
	handle, ok := windowsJobRegistry[pid]
	if ok {
		delete(windowsJobRegistry, pid)
	}
	windowsJobMu.Unlock()
	if ok {
		windows.CloseHandle(handle) //nolint:errcheck
	}
}

// terminateByJob terminates all processes in the job, then cleans up the handle.
// Returns false if no job handle was registered for this PID.
func terminateByJob(pid int) (bool, error) {
	windowsJobMu.RLock()
	handle, ok := windowsJobRegistry[pid]
	windowsJobMu.RUnlock()
	if !ok {
		return false, nil
	}
	err := windows.TerminateJobObject(handle, 1)
	CleanupWindowsJobObject(pid)
	return true, err
}

// jobobjectBasicAccountingInfo mirrors JOBOBJECT_BASIC_ACCOUNTING_INFORMATION
// from the Windows SDK. The struct is not exposed by golang.org/x/sys/windows.
type jobobjectBasicAccountingInfo struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// jobObjectActiveProcesses returns the count of active processes in the job
// associated with the given PID. Returns -1 if no job is registered or on error.
func jobObjectActiveProcesses(pid int) int {
	windowsJobMu.RLock()
	handle, ok := windowsJobRegistry[pid]
	windowsJobMu.RUnlock()
	if !ok {
		return -1
	}

	var info jobobjectBasicAccountingInfo
	err := windows.QueryInformationJobObject(
		handle,
		windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		return -1
	}
	return int(info.ActiveProcesses)
}
