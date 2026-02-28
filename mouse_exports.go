package deskact

import m "github.com/PekingSpades/DeskAct/mouse"

type MouseSettings = m.MouseSettings
type MouseButton = m.MouseButton
type ScrollUnit = m.ScrollUnit
type ScrollDelta = m.ScrollDelta
type MouseCapabilities = m.MouseCapabilities
type MouseOp = m.MouseOp
type MouseError = m.MouseError

var (
	ErrMouseInvalidButton         = m.ErrMouseInvalidButton
	ErrMouseUnsupportedButton     = m.ErrMouseUnsupportedButton
	ErrMouseInvalidScrollUnit     = m.ErrMouseInvalidScrollUnit
	ErrMouseUnsupportedScrollUnit = m.ErrMouseUnsupportedScrollUnit
	ErrMouseActionFailed          = m.ErrMouseActionFailed
)

const (
	DefaultMouseSleep           = m.DefaultMouseSleep
	DefaultMoveSmoothLow        = m.DefaultMoveSmoothLow
	DefaultMoveSmoothHigh       = m.DefaultMoveSmoothHigh
	DefaultMoveSmoothDelay      = m.DefaultMoveSmoothDelay
	DefaultScrollDelay          = m.DefaultScrollDelay
	DefaultScrollSmoothCount    = m.DefaultScrollSmoothCount
	DefaultScrollSmoothInterval = m.DefaultScrollSmoothInterval

	MouseButtonLeft    = m.MouseButtonLeft
	MouseButtonRight   = m.MouseButtonRight
	MouseButtonMiddle  = m.MouseButtonMiddle
	MouseButtonBack    = m.MouseButtonBack
	MouseButtonForward = m.MouseButtonForward
	MouseButtonCenter  = m.MouseButtonCenter

	ScrollUnitLine  = m.ScrollUnitLine
	ScrollUnitPixel = m.ScrollUnitPixel
)

func DefaultMouseSettings() MouseSettings {
	return m.DefaultMouseSettings()
}

func MouseButtonOther(index int) MouseButton {
	return m.MouseButtonOther(index)
}

func CurrentMouseCapabilities() MouseCapabilities {
	return m.CurrentMouseCapabilities()
}

func ScrollDeltaLines(x, y int) ScrollDelta {
	return m.ScrollDeltaLines(x, y)
}

func ScrollDeltaPixels(x, y int) ScrollDelta {
	return m.ScrollDeltaPixels(x, y)
}

func Move(x, y int, settings MouseSettings) error {
	return m.Move(x, y, settings)
}

func Drag(x, y int, button MouseButton, settings MouseSettings) error {
	return m.Drag(x, y, button, settings)
}

func DragSmooth(x, y int, button MouseButton, settings MouseSettings) error {
	return m.DragSmooth(x, y, button, settings)
}

func MoveSmooth(x, y int, settings MouseSettings) error {
	return m.MoveSmooth(x, y, settings)
}

func MoveArgs(x, y int) (int, int) {
	return m.MoveArgs(x, y)
}

func MoveRelative(x, y int, settings MouseSettings) error {
	return m.MoveRelative(x, y, settings)
}

func MoveSmoothRelative(x, y int, settings MouseSettings) error {
	return m.MoveSmoothRelative(x, y, settings)
}

func Location() (int, int) {
	return m.Location()
}

func Click(button MouseButton, double bool, settings MouseSettings) error {
	return m.Click(button, double, settings)
}

func MultiClick(button MouseButton, clickCount int, settings MouseSettings) error {
	return m.MultiClick(button, clickCount, settings)
}

func MoveClick(x, y int, button MouseButton, double bool, settings MouseSettings) error {
	return m.MoveClick(x, y, button, double, settings)
}

func MovesClick(x, y int, button MouseButton, double bool, settings MouseSettings) error {
	return m.MovesClick(x, y, button, double, settings)
}

func Toggle(button MouseButton, down bool, sleepAfter bool, settings MouseSettings) error {
	return m.Toggle(button, down, sleepAfter, settings)
}

func Scroll(delta ScrollDelta, settings MouseSettings) error {
	return m.Scroll(delta, settings)
}

func ScrollLines(x, y int, settings MouseSettings) error {
	return m.ScrollLines(x, y, settings)
}

func ScrollPixels(x, y int, settings MouseSettings) error {
	return m.ScrollPixels(x, y, settings)
}

func ScrollSmooth(delta ScrollDelta, settings MouseSettings) error {
	return m.ScrollSmooth(delta, settings)
}
