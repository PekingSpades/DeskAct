package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"

	deskact "github.com/PekingSpades/DeskAct"
)

const (
	defaultScrollAmount   = 3
	maxDisplayASCIIWidth  = 30
	asciiHeightScale      = 0.5
)

var commandCleaner = strings.NewReplacer(" ", "", "-", "", "_", "")

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Mouse Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")
	fmt.Println("Note: all delays are in milliseconds (ms).")

	reader := bufio.NewReader(os.Stdin)
	dpiSelected, dpiResult := promptDPIInit(reader)
	fmt.Printf("DPI init selected: %v\n", dpiSelected)
	fmt.Printf("DPI init result: %s\n", dpiResult)

	settings := deskact.DefaultMouseSettings()

	for {
		printMenu()
		fmt.Print("Select command: ")
		input := readLine(reader)
		cmd := canonicalize(input)
		if cmd == "" {
			continue
		}
		if cmd == "q" || cmd == "quit" || cmd == "exit" {
			break
		}

		action, ok := normalizeCommand(cmd)
		if !ok {
			fmt.Println("Unknown command.")
			continue
		}

		delay := readNonNegativeIntDefault(reader, "Delay before action (milliseconds, default 0): ", 0)

		switch action {
		case "scroll":
			runScroll(reader, delay, settings)
		case "location":
			runLocation(delay)
		case "move":
			runMove(reader, delay, settings)
		case "left_click":
			runClick(delay, settings, deskact.MouseButtonLeft, 1)
		case "left_double":
			runClick(delay, settings, deskact.MouseButtonLeft, 2)
		case "left_triple":
			runClick(delay, settings, deskact.MouseButtonLeft, 3)
		case "left_drag":
			runDrag(reader, delay, settings, deskact.MouseButtonLeft)
		case "right_click":
			runClick(delay, settings, deskact.MouseButtonRight, 1)
		case "middle_click":
			runClick(delay, settings, deskact.MouseButtonMiddle, 1)
		case "forward_click":
			runClick(delay, settings, deskact.MouseButtonForward, 1)
		case "back_click":
			runClick(delay, settings, deskact.MouseButtonBack, 1)
		default:
			fmt.Println("Unknown command.")
		}
	}

	fmt.Println("Done.")
}

func printMenu() {
	fmt.Println("\nCommands:")
	fmt.Println("  1) scroll")
	fmt.Println("  2) location")
	fmt.Println("  3) move")
	fmt.Println("  4) left click")
	fmt.Println("  5) left double click")
	fmt.Println("  6) left triple click")
	fmt.Println("  7) right click")
	fmt.Println("  8) middle click")
	fmt.Println("  9) forward click")
	fmt.Println("  10) back click")
	fmt.Println("  11) left click drag")
	fmt.Println("  q) exit")
}

func canonicalize(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return ""
	}
	return commandCleaner.Replace(input)
}

func normalizeCommand(cmd string) (string, bool) {
	switch cmd {
	case "1", "scroll":
		return "scroll", true
	case "2", "location", "loc", "pos", "position":
		return "location", true
	case "3", "move":
		return "move", true
	case "4", "left", "leftclick", "left1":
		return "left_click", true
	case "5", "left2", "leftdouble", "leftdoubleclick", "double":
		return "left_double", true
	case "6", "left3", "lefttriple", "lefttripleclick", "triple":
		return "left_triple", true
	case "7", "right", "rightclick", "right1":
		return "right_click", true
	case "8", "middle", "middleclick", "center", "centre":
		return "middle_click", true
	case "9", "forward", "forwardclick", "fwd":
		return "forward_click", true
	case "10", "back", "backclick", "backward", "bwd":
		return "back_click", true
	case "11", "drag", "leftdrag", "leftclickdrag":
		return "left_drag", true
	default:
		return "", false
	}
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptDPIInit(reader *bufio.Reader) (bool, string) {
	for {
		fmt.Print("\nInitialize DPI awareness? (y/n): ")
		input := strings.TrimSpace(strings.ToLower(readLine(reader)))
		switch input {
		case "y", "yes":
			result := deskact.InitDPIAwareness()
			return true, fmt.Sprintf("%v", result)
		case "n", "no":
			return false, "skipped"
		default:
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}

func readStringDefault(reader *bufio.Reader, prompt, defaultVal string) string {
	for {
		fmt.Print(prompt)
		line := readLine(reader)
		if line == "" {
			return defaultVal
		}
		return line
	}
}

func readNonNegativeIntDefault(reader *bufio.Reader, prompt string, defaultVal int) int {
	for {
		fmt.Print(prompt)
		line := readLine(reader)
		if line == "" {
			return defaultVal
		}
		value, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("Please enter a valid integer.")
			continue
		}
		if value < 0 {
			fmt.Println("Please enter a non-negative integer.")
			continue
		}
		return value
	}
}

func readInt(reader *bufio.Reader, prompt string) int {
	for {
		fmt.Print(prompt)
		line := readLine(reader)
		if line == "" {
			fmt.Println("Please enter a value.")
			continue
		}
		value, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("Please enter a valid integer.")
			continue
		}
		return value
	}
}

func readIntInRange(reader *bufio.Reader, prompt string, minValue, maxValue int) int {
	for {
		value := readInt(reader, prompt)
		if value < minValue || value > maxValue {
			fmt.Printf("Please enter a value between %d and %d.\n", minValue, maxValue)
			continue
		}
		return value
	}
}

func waitDelay(delay int) {
	if delay <= 0 {
		return
	}
	fmt.Printf("Waiting %d ms...\n", delay)
	deskact.MilliSleep(delay)
}

func readScrollUnit(reader *bufio.Reader) (deskact.ScrollUnit, string) {
	for {
		unit := strings.ToLower(readStringDefault(reader, "Scroll unit (line/pixel, default line): ", "line"))
		switch unit {
		case "line", "lines", "l":
			return deskact.ScrollUnitLine, "line"
		case "pixel", "pixels", "p":
			return deskact.ScrollUnitPixel, "pixel"
		default:
			fmt.Println("Please enter line or pixel.")
		}
	}
}

func readScrollDirection(reader *bufio.Reader) (string, int, int) {
	for {
		direction := strings.ToLower(readStringDefault(reader, "Direction (up/down/left/right, default down): ", "down"))
		switch direction {
		case "up", "u":
			return "up", 0, 1
		case "down", "d":
			return "down", 0, -1
		case "left", "l":
			return "left", -1, 0
		case "right", "r":
			return "right", 1, 0
		default:
			fmt.Println("Please enter up, down, left, or right.")
		}
	}
}

func runScroll(reader *bufio.Reader, delay int, settings deskact.MouseSettings) {
	unit, unitLabel := readScrollUnit(reader)
	direction, dx, dy := readScrollDirection(reader)
	amount := readNonNegativeIntDefault(reader, "Amount per unit (default 3): ", defaultScrollAmount)
	if amount == 0 {
		fmt.Println("Scroll amount is 0; skipping.")
		return
	}

	dx *= amount
	dy *= amount

	waitDelay(delay)
	err := deskact.Scroll(deskact.ScrollDelta{X: dx, Y: dy, Unit: unit}, settings)
	if err != nil {
		fmt.Printf("Scroll error: %v\n", err)
		return
	}
	fmt.Printf("Scrolled %s by %d %s(s).\n", direction, amount, unitLabel)
}

func runLocation(delay int) {
	waitDelay(delay)
	x, y := deskact.Location()
	fmt.Printf("Mouse location: (%d, %d)\n", x, y)

	displayOptions := deskact.DefaultDisplayOptions()
	displays := deskact.AllDisplays(displayOptions)
	if len(displays) == 0 {
		fmt.Println("No displays detected.")
		return
	}

	layoutX, layoutY := x, y
	var hitDisplay *deskact.Display
	var hitInfo deskact.DisplayInfo
	var relLogicalX, relLogicalY int
	var percentX, percentY float64

	for _, d := range displays {
		if !d.Contains(x, y) {
			continue
		}
		hitDisplay = d
		hitInfo = d.Info()
		relPhysX, relPhysY, ok := d.ToRelative(x, y)
		if !ok {
			break
		}
		relLogicalX, relLogicalY, percentX, percentY = mapRelativeToOrigin(
			relPhysX, relPhysY,
			hitInfo.Size.W, hitInfo.Size.H,
			hitInfo.Origin.W, hitInfo.Origin.H,
		)
		layoutX = hitInfo.Origin.X + relLogicalX
		layoutY = hitInfo.Origin.Y + relLogicalY
		break
	}

	if hitDisplay != nil {
		fmt.Printf("On display #%d (main=%v) at (%d, %d) of %dx%d (%.2f%%, %.2f%%)\n",
			hitInfo.Index, hitInfo.IsMain, relLogicalX, relLogicalY,
			hitInfo.Origin.W, hitInfo.Origin.H,
			percentX*100, percentY*100)
	} else {
		fmt.Println("Mouse is outside all display bounds.")
	}

	diagram := buildDisplayDiagram(displays, layoutX, layoutY, hitDisplay != nil)
	if diagram != "" {
		fmt.Println("Display layout (max display width scaled to 30 chars):")
		fmt.Println(diagram)
		fmt.Println("Legend: @ = mouse, * = main display")
	}
}

func formatDisplayRange(info deskact.DisplayInfo) string {
	if info.Size.W <= 0 || info.Size.H <= 0 {
		return fmt.Sprintf("size=%dx%d range: unavailable", info.Size.W, info.Size.H)
	}
	return fmt.Sprintf("size=%dx%d range: x=0..%d, y=0..%d",
		info.Size.W, info.Size.H, info.Size.W-1, info.Size.H-1)
}

func selectDisplay(reader *bufio.Reader) (*deskact.Display, deskact.DisplayInfo) {
	displayOptions := deskact.DefaultDisplayOptions()
	displays := deskact.AllDisplays(displayOptions)
	if len(displays) == 0 {
		fmt.Println("No displays detected.")
		return nil, deskact.DisplayInfo{}
	}

	displayMap := make(map[int]*deskact.Display, len(displays))
	fmt.Println("Available displays (physical pixel ranges):")
	for _, d := range displays {
		info := d.Info()
		displayMap[info.Index] = d
		fmt.Printf("  #%d (main=%v) %s\n", info.Index, info.IsMain, formatDisplayRange(info))
	}

	for {
		index := readInt(reader, "Select display index: ")
		d, ok := displayMap[index]
		if !ok {
			fmt.Println("Invalid display index.")
			continue
		}
		return d, d.Info()
	}
}

func promptPhysicalPoint(reader *bufio.Reader, info deskact.DisplayInfo, label string) (int, int, bool) {
	if info.Size.W <= 0 || info.Size.H <= 0 {
		fmt.Println("Selected display has invalid physical size.")
		return 0, 0, false
	}
	maxX := info.Size.W - 1
	maxY := info.Size.H - 1
	x := readIntInRange(reader, fmt.Sprintf("%s X (0..%d): ", label, maxX), 0, maxX)
	y := readIntInRange(reader, fmt.Sprintf("%s Y (0..%d): ", label, maxY), 0, maxY)
	return x, y, true
}

func runMove(reader *bufio.Reader, delay int, settings deskact.MouseSettings) {
	display, info := selectDisplay(reader)
	if display == nil {
		return
	}
	x, y, ok := promptPhysicalPoint(reader, info, "Target")
	if !ok {
		return
	}
	waitDelay(delay)
	if err := display.Move(x, y, settings); err != nil {
		fmt.Printf("Move error: %v\n", err)
		return
	}
	fmt.Printf("Moved to display #%d at (%d, %d)\n", info.Index, x, y)
}

func runDrag(reader *bufio.Reader, delay int, settings deskact.MouseSettings, button deskact.MouseButton) {
	display, info := selectDisplay(reader)
	if display == nil {
		return
	}
	startX, startY, ok := promptPhysicalPoint(reader, info, "Start")
	if !ok {
		return
	}
	endX, endY, ok := promptPhysicalPoint(reader, info, "End")
	if !ok {
		return
	}
	waitDelay(delay)
	if err := display.Drag(startX, startY, endX, endY, button, settings); err != nil {
		fmt.Printf("Drag error: %v\n", err)
		return
	}
	fmt.Printf("Dragged %s on display #%d from (%d, %d) to (%d, %d)\n",
		button, info.Index, startX, startY, endX, endY)
}

func runClick(delay int, settings deskact.MouseSettings, button deskact.MouseButton, count int) {
	if count < 1 {
		count = 1
	}
	waitDelay(delay)
	if err := deskact.MultiClick(button, count, settings); err != nil {
		fmt.Printf("Click error: %v\n", err)
		return
	}
	fmt.Printf("Clicked %s x%d\n", button, count)
}

func mapRelativeToOrigin(relPhysX, relPhysY, physW, physH, originW, originH int) (relLogicalX, relLogicalY int, percentX, percentY float64) {
	percentX = safeRatio(relPhysX, physW)
	percentY = safeRatio(relPhysY, physH)
	relLogicalX = int(math.Round(percentX * float64(originW)))
	relLogicalY = int(math.Round(percentY * float64(originH)))
	if originW > 0 {
		relLogicalX = clampInt(relLogicalX, 0, originW-1)
	}
	if originH > 0 {
		relLogicalY = clampInt(relLogicalY, 0, originH-1)
	}
	return relLogicalX, relLogicalY, percentX, percentY
}

func buildDisplayDiagram(displays []*deskact.Display, mouseX, mouseY int, showMouse bool) string {
	if len(displays) == 0 {
		return ""
	}

	infos := make([]deskact.DisplayInfo, 0, len(displays))
	maxDisplayWidth := 0
	minX, minY := 0, 0

	for i, d := range displays {
		info := d.Info()
		infos = append(infos, info)
		if i == 0 {
			minX = info.Origin.X
			minY = info.Origin.Y
		} else {
			if info.Origin.X < minX {
				minX = info.Origin.X
			}
			if info.Origin.Y < minY {
				minY = info.Origin.Y
			}
		}
		if info.Origin.W > maxDisplayWidth {
			maxDisplayWidth = info.Origin.W
		}
	}

	if maxDisplayWidth <= 0 {
		return ""
	}

	scaleX := 1.0
	if maxDisplayWidth > maxDisplayASCIIWidth {
		scaleX = float64(maxDisplayASCIIWidth) / float64(maxDisplayWidth)
	}
	scaleY := scaleX * asciiHeightScale

	type scaledDisplay struct {
		info     deskact.DisplayInfo
		x0, y0   int
		width    int
		height   int
	}

	scaledDisplays := make([]scaledDisplay, 0, len(infos))
	maxGridX, maxGridY := 0, 0

	for _, info := range infos {
		if info.Origin.W <= 0 || info.Origin.H <= 0 {
			continue
		}
		w := int(math.Round(float64(info.Origin.W) * scaleX))
		h := int(math.Round(float64(info.Origin.H) * scaleY))
		if w < 2 {
			w = 2
		}
		if h < 2 {
			h = 2
		}
		x0 := int(math.Round(float64(info.Origin.X-minX) * scaleX))
		y0 := int(math.Round(float64(info.Origin.Y-minY) * scaleY))
		x1 := x0 + w - 1
		y1 := y0 + h - 1
		if x1 > maxGridX {
			maxGridX = x1
		}
		if y1 > maxGridY {
			maxGridY = y1
		}
		scaledDisplays = append(scaledDisplays, scaledDisplay{
			info:   info,
			x0:     x0,
			y0:     y0,
			width:  w,
			height: h,
		})
	}

	if len(scaledDisplays) == 0 {
		return ""
	}

	gridWidth := maxGridX + 1
	gridHeight := maxGridY + 1
	grid := make([][]byte, gridHeight)
	for y := range grid {
		row := make([]byte, gridWidth)
		for x := range row {
			row[x] = ' '
		}
		grid[y] = row
	}

	for _, d := range scaledDisplays {
		label := displayLabel(d.info)
		drawDisplayRect(grid, d.x0, d.y0, d.width, d.height, label)
	}

	if showMouse {
		mouseGX := int(math.Round(float64(mouseX-minX) * scaleX))
		mouseGY := int(math.Round(float64(mouseY-minY) * scaleY))
		mouseGX = clampInt(mouseGX, 0, gridWidth-1)
		mouseGY = clampInt(mouseGY, 0, gridHeight-1)
		setGridCell(grid, mouseGX, mouseGY, '@')
	}

	var builder strings.Builder
	for i, row := range grid {
		line := strings.TrimRight(string(row), " ")
		builder.WriteString(line)
		if i < len(grid)-1 {
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func displayLabel(info deskact.DisplayInfo) string {
	if info.IsMain {
		return fmt.Sprintf("%d*", info.Index)
	}
	return fmt.Sprintf("%d", info.Index)
}

func drawDisplayRect(grid [][]byte, x0, y0, w, h int, label string) {
	if w < 2 || h < 2 {
		return
	}
	x1 := x0 + w - 1
	y1 := y0 + h - 1

	for y := y0 + 1; y < y1; y++ {
		for x := x0 + 1; x < x1; x++ {
			if getGridCell(grid, x, y) == ' ' {
				setGridCell(grid, x, y, '.')
			}
		}
	}

	for x := x0; x <= x1; x++ {
		setGridCell(grid, x, y0, '-')
		setGridCell(grid, x, y1, '-')
	}
	for y := y0; y <= y1; y++ {
		setGridCell(grid, x0, y, '|')
		setGridCell(grid, x1, y, '|')
	}
	setGridCell(grid, x0, y0, '+')
	setGridCell(grid, x1, y0, '+')
	setGridCell(grid, x0, y1, '+')
	setGridCell(grid, x1, y1, '+')

	if label == "" {
		return
	}
	labelX := x0 + 1
	labelY := y0 + 1
	if labelX > x1-1 || labelY > y1-1 {
		return
	}
	if labelX+len(label)-1 > x1-1 {
		return
	}
	for i := 0; i < len(label); i++ {
		setGridCell(grid, labelX+i, labelY, label[i])
	}
}

func setGridCell(grid [][]byte, x, y int, ch byte) {
	if y < 0 || y >= len(grid) {
		return
	}
	row := grid[y]
	if x < 0 || x >= len(row) {
		return
	}
	row[x] = ch
}

func getGridCell(grid [][]byte, x, y int) byte {
	if y < 0 || y >= len(grid) {
		return ' '
	}
	row := grid[y]
	if x < 0 || x >= len(row) {
		return ' '
	}
	return row[x]
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func safeRatio(value, size int) float64 {
	if size <= 0 {
		return 0
	}
	ratio := float64(value) / float64(size)
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}
