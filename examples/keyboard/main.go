package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	deskact "github.com/PekingSpades/DeskAct"
)

const (
	chineseUnit = 2
	englishUnit = 1
	emojiUnit   = 1
)

var commandCleaner = strings.NewReplacer(" ", "", "-", "", "_", "")

type category struct {
	name   string
	length int
	runes  []rune
}

var (
	chineseRunes = []rune{
		'\u4f60', '\u597d', '\u4e16', '\u754c', '\u4e2d',
		'\u56fd', '\u6587', '\u5b57', '\u6d4b', '\u8bd5',
	}
	englishRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	emojiRunes   = []rune{
		'\U0001F600', '\U0001F60A', '\U0001F44D',
		'\U0001F680', '\U0001F389', '\U0001F31F',
	}
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Keyboard Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")
	fmt.Println("Note: input is sent to the active window.")

	reader := bufio.NewReader(os.Stdin)
	settings := deskact.DefaultKeyboardSettings()

	for {
		printMenu()
		fmt.Print("Select command: ")
		cmd := canonicalize(readLine(reader))
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

		switch action {
		case "type":
			runType(reader, settings)
		case "state":
			runState()
		case "list_keys":
			runListKeys()
		case "delay_tap":
			runDelayTap(reader, settings)
		default:
			fmt.Println("Unknown command.")
		}
	}

	fmt.Println("Done.")
}

func printMenu() {
	fmt.Println("\nCommands:")
	fmt.Println("  1) type")
	fmt.Println("  2) state")
	fmt.Println("  3) list supported keys")
	fmt.Println("  4) delay key tap")
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
	case "1", "type":
		return "type", true
	case "2", "state", "status", "keyboardstate":
		return "state", true
	case "3", "keys", "listkeys", "supported", "supportedkeys":
		return "list_keys", true
	case "4", "tap", "delaytap", "delay", "keytap":
		return "delay_tap", true
	default:
		return "", false
	}
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func readYesNo(reader *bufio.Reader, prompt string) bool {
	for {
		fmt.Print(prompt)
		input := strings.TrimSpace(strings.ToLower(readLine(reader)))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter y or n.")
		}
	}
}

func readIntMin(reader *bufio.Reader, prompt string, minValue int) int {
	for {
		fmt.Print(prompt)
		line := readLine(reader)
		if line == "" {
			fmt.Printf("Please enter a value >= %d.\n", minValue)
			continue
		}
		value, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("Please enter a valid integer.")
			continue
		}
		if value < minValue {
			fmt.Printf("Please enter a value >= %d.\n", minValue)
			continue
		}
		return value
	}
}

func runType(reader *bufio.Reader, settings deskact.KeyboardSettings) {
	includeChinese := readYesNo(reader, "Include Chinese? (y/n): ")
	includeEnglish := readYesNo(reader, "Include English? (y/n): ")
	includeEmoji := readYesNo(reader, "Include Emoji? (y/n): ")
	if !includeChinese && !includeEnglish && !includeEmoji {
		fmt.Println("At least one category must be selected.")
		return
	}

	lengthUnits := readIntMin(reader, "Length (Chinese=2, English=1, Emoji=1): ", 1)
	delayMs := readIntMin(reader, "Delay before typing (ms): ", 0)

	cats := buildCategories(includeChinese, includeEnglish, includeEmoji)
	if includeChinese && !includeEnglish && !includeEmoji && lengthUnits%2 == 1 {
		fmt.Printf("Length %d adjusted to %d because Chinese counts as 2.\n", lengthUnits, lengthUnits+1)
		lengthUnits++
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	text, units, err := buildText(lengthUnits, cats, rng)
	if err != nil {
		fmt.Printf("Failed to build text: %v\n", err)
		return
	}

	fmt.Printf("Generated text (units=%d): %s\n", units, text)
	waitDelay(delayMs)
	deskact.Type(text, 0, settings)
	fmt.Println("Type done.")
}

func buildCategories(includeChinese, includeEnglish, includeEmoji bool) []category {
	cats := make([]category, 0, 3)
	if includeChinese {
		cats = append(cats, category{name: "chinese", length: chineseUnit, runes: chineseRunes})
	}
	if includeEnglish {
		cats = append(cats, category{name: "english", length: englishUnit, runes: englishRunes})
	}
	if includeEmoji {
		cats = append(cats, category{name: "emoji", length: emojiUnit, runes: emojiRunes})
	}
	return cats
}

func buildText(target int, cats []category, rng *rand.Rand) (string, int, error) {
	if target <= 0 {
		return "", 0, fmt.Errorf("length must be > 0")
	}
	if len(cats) == 0 {
		return "", 0, fmt.Errorf("no categories selected")
	}

	var b strings.Builder
	remaining := target

	for remaining > 0 {
		var fit []category
		for _, c := range cats {
			if c.length <= remaining {
				fit = append(fit, c)
			}
		}
		if len(fit) == 0 {
			return b.String(), target - remaining, fmt.Errorf("cannot satisfy length %d with current selection", target)
		}

		cat := fit[rng.Intn(len(fit))]
		r := cat.runes[rng.Intn(len(cat.runes))]
		b.WriteRune(r)
		remaining -= cat.length
	}

	return b.String(), target, nil
}

func waitDelay(delay int) {
	if delay <= 0 {
		return
	}
	fmt.Printf("Waiting %d ms...\n", delay)
	deskact.MilliSleep(delay)
}

func waitDelaySeconds(seconds int) {
	if seconds <= 0 {
		return
	}
	fmt.Printf("Waiting %d second(s)...\n", seconds)
	deskact.MilliSleep(seconds * 1000)
}

func runState() {
	snapshot, err := deskact.KeyboardStateCurrent()
	if err != nil {
		fmt.Printf("Keyboard state error: %v\n", err)
		return
	}
	fmt.Println("Keyboard state:")
	fmt.Printf("  Shift: %s\n", pressStateLabel(snapshot.Shift))
	fmt.Printf("  Ctrl: %s\n", pressStateLabel(snapshot.Ctrl))
	fmt.Printf("  Alt: %s\n", pressStateLabel(snapshot.Alt))
	fmt.Printf("  Cmd: %s\n", pressStateLabel(snapshot.Cmd))
	fmt.Printf("  CapsLock: %s\n", toggleStateLabel(snapshot.CapsLock))
	fmt.Printf("  NumLock: %s\n", toggleStateLabel(snapshot.NumLock))
	fmt.Printf("  ScrollLock: %s\n", toggleStateLabel(snapshot.ScrollLock))
}

func runListKeys() {
	keys := deskact.SupportedKeyNames()
	if len(keys) == 0 {
		fmt.Println("No supported keys reported for this platform.")
		return
	}
	fmt.Printf("Supported key names (%d):\n", len(keys))
	printWrappedList(keys, 90)
}

func runDelayTap(reader *bufio.Reader, settings deskact.KeyboardSettings) {
	keys := deskact.SupportedKeyNames()
	if len(keys) == 0 {
		fmt.Println("No supported keys reported for this platform.")
		return
	}
	fmt.Println("Supported key names:")
	printWrappedList(keys, 90)

	seconds := readIntMin(reader, "Delay before key tap (seconds): ", 0)
	fmt.Print("Key to tap: ")
	key := normalizeKeyInput(readLine(reader))
	if key == "" {
		fmt.Println("Key cannot be empty.")
		return
	}

	waitDelaySeconds(seconds)
	if err := deskact.KeyTap(key, nil, settings); err != nil {
		fmt.Printf("Key tap error: %v\n", err)
		return
	}
	fmt.Println("Key tap done.")
}

func normalizeKeyInput(key string) string {
	if key == "" {
		return ""
	}
	if len([]rune(key)) == 1 {
		return key
	}
	return strings.ToLower(key)
}

func pressStateLabel(state deskact.KeyboardPressState) string {
	switch state {
	case deskact.KeyboardPressUp:
		return "up"
	case deskact.KeyboardPressDown:
		return "down"
	case deskact.KeyboardPressUnsupported:
		return "unsupported"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}

func toggleStateLabel(state deskact.KeyboardToggleState) string {
	switch state {
	case deskact.KeyboardToggleOff:
		return "off"
	case deskact.KeyboardToggleOn:
		return "on"
	case deskact.KeyboardToggleUnsupported:
		return "unsupported"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}

func printWrappedList(items []string, maxLine int) {
	if maxLine < 20 {
		maxLine = 20
	}
	var line strings.Builder
	for _, item := range items {
		if line.Len() == 0 {
			line.WriteString(item)
			continue
		}
		if line.Len()+2+len(item) > maxLine {
			fmt.Println(line.String())
			line.Reset()
			line.WriteString(item)
			continue
		}
		line.WriteString(", ")
		line.WriteString(item)
	}
	if line.Len() > 0 {
		fmt.Println(line.String())
	}
}
