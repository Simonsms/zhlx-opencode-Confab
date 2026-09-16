//go:build windows

package process

import (
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// stillActive is the value GetExitCodeProcess returns for a process that has
// not exited (Win32 STILL_ACTIVE = 259). x/sys/windows does not export it.
const stillActive uint32 = 0x103

// platformIsRunning checks liveness with OpenProcess + GetExitCodeProcess.
// Windows has no signal-0 probe: os.Process.Signal only supports os.Kill, so
// the liveness test must read the process's exit code directly. A process is
// running iff its exit code is still STILL_ACTIVE.
func platformIsRunning(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var exitCode uint32
	if err := windows.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}

// procSnapshotEntry is the subset of a Toolhelp32 PROCESSENTRY32 record the
// package needs.
type procSnapshotEntry struct {
	name string
	ppid int
}

// snapshotProc entries every live process via CreateToolhelp32Snapshot and
// returns a map keyed by PID (or nil if the snapshot itself failed).
func snapshotProc() map[uint32]procSnapshotEntry {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil
	}
	procs := make(map[uint32]procSnapshotEntry)
	for {
		procs[entry.ProcessID] = procSnapshotEntry{
			name: windows.UTF16ToString(entry.ExeFile[:]),
			ppid: int(entry.ParentProcessID),
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return procs
}

// findProc returns the snapshot entry for a PID, or ok=false when the
// process doesn't exist (or the snapshot failed).
func findProc(pid int) (procSnapshotEntry, bool) {
	procs := snapshotProc()
	if procs == nil {
		return procSnapshotEntry{}, false
	}
	entry, ok := procs[uint32(pid)]
	return entry, ok
}

// platformName returns the executable basename from the snapshot. ExeFile is
// already a basename; the filepath.Base is defensive.
func platformName(pid int) string {
	entry, ok := findProc(pid)
	if !ok {
		return ""
	}
	return filepath.Base(entry.name)
}

// platformParentPID returns ParentProcessID from the snapshot.
func platformParentPID(pid int) int {
	entry, ok := findProc(pid)
	if !ok {
		return 0
	}
	return entry.ppid
}

// platformCmdline falls back to the executable name: reading the full command
// line of an arbitrary Windows process requires targeting its PEB via
// NtQueryInformationProcess + ReadProcessMemory, which is out of proportion
// for a best-effort parent detector. Provider name regexes (`\bclaude\b`,
// `\bcodex\b`) still match the basename.
func platformCmdline(pid int) string {
	return platformName(pid)
}
