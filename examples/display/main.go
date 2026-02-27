package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	deskact "github.com/PekingSpades/DeskAct"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Display Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")

	dpiSelected, dpiResult := promptDPIInit()
	fmt.Printf("DPI init selected: %v\n", dpiSelected)
	fmt.Printf("DPI init result: %s\n", dpiResult)

	displayOptions := deskact.DefaultDisplayOptions()
	mouseSettings := deskact.DefaultMouseSettings()

	count := deskact.DisplayCount()
	fmt.Printf("\nTotal displays: %d\n", count)
	fmt.Println("----------------------------------------")

	displays := deskact.AllDisplays(displayOptions)
	for _, d := range displays {
		info := d.Info()
		fmt.Printf("Display #%d\n", info.Index)
		fmt.Printf("  ID:         %d\n", info.ID)
		fmt.Printf("  ElectronID: %d\n", info.ElectronID)
		fmt.Printf("  IsMain:     %v\n", info.IsMain)
		fmt.Printf("  Origin:     {X: %d, Y: %d, W: %d, H: %d}\n",
			info.Origin.X, info.Origin.Y, info.Origin.W, info.Origin.H)
		fmt.Printf("  Size:       {W: %d, H: %d}\n", info.Size.W, info.Size.H)
		fmt.Printf("  Scale:      %.2f\n", info.ScaleFactor)
		fmt.Println("----------------------------------------")
	}

	mainDisplay := deskact.MainDisplay(displayOptions)
	if mainDisplay != nil {
		fmt.Printf("MainDisplay: Index=%d, Size=%dx%d, Scale=%.2f\n",
			mainDisplay.Index(), mainDisplay.Width(), mainDisplay.Height(), mainDisplay.Scale())
	}

	fmt.Println("\n========================================")
	fmt.Println("Moving mouse to each display's corners")
	fmt.Println("========================================")

	margin := 32

	for _, d := range displays {
		info := d.Info()
		w, h := info.Size.W, info.Size.H
		fmt.Printf("\nDisplay #%d (%dx%d) @ (%d,%d):\n",
			info.Index, w, h, info.Origin.X, info.Origin.Y)

		positions := []struct {
			name string
			x, y int
		}{
			{"top-left", margin, margin},
			{"top-right", w - 1 - margin, margin},
			{"bottom-left", margin, h - 1 - margin},
			{"bottom-right", w - 1 - margin, h - 1 - margin},
			{"center", w / 2, h / 2},
		}

		for _, pos := range positions {
			if err := d.Move(pos.x, pos.y, mouseSettings); err != nil {
				fmt.Printf("  %-12s: move error: %v\n", pos.name, err)
				continue
			}
			deskact.MilliSleep(800)

			absX, absY := deskact.Location()
			relX, relY, ok := d.MouseLocation()
			if ok {
				fmt.Printf("  %-12s: target=(%4d,%4d) abs=(%5d,%5d) rel=(%4d,%4d)\n",
					pos.name, pos.x, pos.y, absX, absY, relX, relY)
			} else {
				fmt.Printf("  %-12s: target=(%4d,%4d) abs=(%5d,%5d) (mouse not on this display)\n",
					pos.name, pos.x, pos.y, absX, absY)
			}
			deskact.MilliSleep(200)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Done!")
	fmt.Println("========================================")

	var logBuilder strings.Builder
	logBuilder.WriteString("DeskAct Display Example\n")
	logBuilder.WriteString(fmt.Sprintf("Go version: %s\n", runtime.Version()))
	logBuilder.WriteString(fmt.Sprintf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	logBuilder.WriteString(fmt.Sprintf("DPI init selected: %v\n", dpiSelected))
	logBuilder.WriteString(fmt.Sprintf("DPI init result: %s\n", dpiResult))
	logBuilder.WriteString(fmt.Sprintf("Total displays: %d\n", count))
	for _, d := range displays {
		info := d.Info()
		logBuilder.WriteString(fmt.Sprintf("Display #%d: ID=%d, ElectronID=%d, IsMain=%v, Origin=(%d,%d,%d,%d), Size=%dx%d, Scale=%.2f\n",
			info.Index, info.ID, info.ElectronID, info.IsMain, info.Origin.X, info.Origin.Y, info.Origin.W, info.Origin.H, info.Size.W, info.Size.H, info.ScaleFactor))
	}

	fmt.Println("\nPress 's' to save log and exit, or 'e' to exit directly:")
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "s" {
			if err := os.WriteFile("display_log.txt", []byte(logBuilder.String()), 0644); err != nil {
				fmt.Printf("Failed to save log: %v\n", err)
			} else {
				fmt.Println("Log saved to display_log.txt")
			}
			break
		} else if input == "e" {
			break
		}
		fmt.Println("Press 's' to save log and exit, or 'e' to exit directly:")
	}
}

func promptDPIInit() (bool, string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nInitialize DPI awareness? (y/n): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
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
