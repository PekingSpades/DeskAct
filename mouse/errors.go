package mouse

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"syscall"
)

var (
	ErrMouseInvalidButton         = errors.New("invalid mouse button")
	ErrMouseUnsupportedButton     = errors.New("mouse button not supported")
	ErrMouseInvalidScrollUnit     = errors.New("invalid scroll unit")
	ErrMouseUnsupportedScrollUnit = errors.New("scroll unit not supported")
	ErrMouseActionFailed          = errors.New("mouse action failed")
)

// MouseOp describes a mouse operation for error reporting.
type MouseOp string

const (
	MouseOpClick      MouseOp = "click"
	MouseOpMultiClick MouseOp = "multiClick"
	MouseOpToggle     MouseOp = "toggle"
	MouseOpMove       MouseOp = "move"
	MouseOpMoveSmooth MouseOp = "moveSmooth"
	MouseOpDrag       MouseOp = "drag"
	MouseOpScroll     MouseOp = "scroll"
)

// MouseError wraps mouse operation failures with context.
type MouseError struct {
	Op         MouseOp
	Button     MouseButton
	ClickCount int
	Unit       ScrollUnit
	Code       int
	Detail     string
	Cause      error
}

func (e *MouseError) Error() string {
	if e == nil {
		return "<nil>"
	}
	var b strings.Builder
	b.WriteString("mouse ")
	if e.Op != "" {
		b.WriteString(string(e.Op))
	} else {
		b.WriteString("error")
	}
	if e.Button != 0 {
		b.WriteString(" ")
		b.WriteString(e.Button.String())
	}
	if e.ClickCount > 0 {
		b.WriteString(fmt.Sprintf(" count=%d", e.ClickCount))
	}
	if e.Op == MouseOpScroll {
		b.WriteString(" unit=")
		b.WriteString(e.Unit.String())
	}
	detail := e.Detail
	if detail == "" && e.Cause != nil {
		detail = e.Cause.Error()
	}
	if detail != "" {
		b.WriteString(": ")
		b.WriteString(detail)
	}
	if e.Code != 0 {
		b.WriteString(fmt.Sprintf(" (code=%d)", e.Code))
	}
	return b.String()
}

func (e *MouseError) Unwrap() error {
	return e.Cause
}

func wrapMouseError(op MouseOp, cause error, button MouseButton, unit ScrollUnit, clickCount int, detail string, code int) error {
	return &MouseError{
		Op:         op,
		Button:     button,
		ClickCount: clickCount,
		Unit:       unit,
		Code:       code,
		Detail:     detail,
		Cause:      cause,
	}
}

func mouseActionError(op MouseOp, button MouseButton, clickCount int, code int) error {
	return wrapMouseError(op, ErrMouseActionFailed, button, 0, clickCount, mouseErrorDetail(code), code)
}

func mouseErrorDetail(code int) string {
	if code == 0 {
		return ""
	}

	switch runtime.GOOS {
	case "windows":
		return syscall.Errno(code).Error()
	case "darwin":
		cgErrors := map[int]string{
			0:    "kCGErrorSuccess",
			1000: "kCGErrorFailure",
			1001: "kCGErrorIllegalArgument",
			1002: "kCGErrorInvalidConnection",
			1003: "kCGErrorInvalidContext",
			1004: "kCGErrorCannotComplete",
			1005: "kCGErrorNotImplemented",
			1006: "kCGErrorRangeCheck",
			1007: "kCGErrorTypeCheck",
			1008: "kCGErrorNoCurrentPoint",
			1010: "kCGErrorInvalidOperation",
		}
		if v, ok := cgErrors[code]; ok {
			return v
		}
	default:
		if code == 1 {
			return "XTestFakeButtonEvent returned false"
		}
	}

	return fmt.Sprintf("code=%d", code)
}
