//go:build windows

package mouse

/*
#cgo LDFLAGS: -luser32
#include <windows.h>
#include <stdint.h>

typedef struct {
    HWND  hwnd;
    DWORD pid;
} pid_to_hwnd_state;

static BOOL CALLBACK pid_to_hwnd_cb(HWND hwnd, LPARAM lparam) {
    pid_to_hwnd_state *s = (pid_to_hwnd_state*)lparam;
    DWORD pid = 0;
    GetWindowThreadProcessId(hwnd, &pid);
    if (pid == s->pid) {
        // Prefer top-level visible windows. Skip tool windows / invisible.
        if (IsWindowVisible(hwnd)) {
            s->hwnd = hwnd;
            return FALSE;
        }
        if (s->hwnd == NULL) {
            s->hwnd = hwnd;
        }
    }
    return TRUE;
}

static uintptr_t hwnd_by_pid(unsigned int pid) {
    pid_to_hwnd_state s = {NULL, (DWORD)pid};
    EnumWindows(pid_to_hwnd_cb, (LPARAM)&s);
    return (uintptr_t)s.hwnd;
}
*/
import "C"

func hwndByPIDPlatform(pid int) uint64 {
	if pid <= 0 {
		return 0
	}
	return uint64(C.hwnd_by_pid(C.uint(pid)))
}

// xidByPIDPlatform is X11-only; on Windows MoveWithPID never calls it.
func xidByPIDPlatform(pid int) uint64 { return 0 }

// frontWindowIDByPIDPlatform is macOS-only; on Windows MoveWithPID never
// calls it (hwndByPIDPlatform covers the Windows path).
func frontWindowIDByPIDPlatform(pid int) uint64 { return 0 }
