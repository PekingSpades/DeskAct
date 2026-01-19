package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	deskact "github.com/PekingSpades/DeskAct"
)

const defaultScrollAmount = 3

var commandCleaner = strings.NewReplacer(" ", "", "-", "", "_", "")

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Mouse Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")
	fmt.Println("Note: all delays are in milliseconds (ms).")

	settings := deskact.DefaultMouseSettings()
	reader := bufio.NewReader(os.Stdin)

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
	default:
		return "", false
	}
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
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
}

func runMove(reader *bufio.Reader, delay int, settings deskact.MouseSettings) {
	x := readInt(reader, "X: ")
	y := readInt(reader, "Y: ")
	waitDelay(delay)
	if err := deskact.Move(x, y, settings); err != nil {
		fmt.Printf("Move error: %v\n", err)
		return
	}
	fmt.Printf("Moved to (%d, %d)\n", x, y)
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
