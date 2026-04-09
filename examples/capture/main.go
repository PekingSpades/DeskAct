package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"strconv"
	"strings"

	deskact "github.com/PekingSpades/DeskAct"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Capture Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")
	dpiSelected, dpiResult := promptDPIInit()
	fmt.Printf("DPI init selected: %v\n", dpiSelected)
	fmt.Printf("DPI init result: %s\n", dpiResult)
	displayOptions := deskact.DefaultDisplayOptions()
	captureOptions := deskact.DefaultCaptureOptions()
	backendSelected, backendResult := promptCaptureBackend(captureOptions.Backend)
	captureOptions.Backend = backendResult
	fmt.Printf("Capture backend selected: %v\n", backendSelected)
	fmt.Printf("Capture backend result: %s\n", captureBackendLabel(captureOptions.Backend))
	if runtime.GOOS == "darwin" {
		captureOptions.ExcludedWindowIDs = promptExcludedWindowIDs(captureOptions.Backend)
		fmt.Printf("Excluded window IDs: %s\n", formatExcludedWindowIDs(captureOptions.ExcludedWindowIDs))
	}

	displays := deskact.AllDisplays(displayOptions)
	count := len(displays)
	fmt.Printf("\nTotal displays: %d\n", count)
	fmt.Println("----------------------------------------")

	for _, d := range displays {
		info := d.Info()
		fmt.Printf("Display #%d\n", info.Index)
		fmt.Printf("  ID:       %d\n", info.ID)
		fmt.Printf("  IsMain:   %v\n", info.IsMain)
		fmt.Printf("  Origin:   {X: %d, Y: %d, W: %d, H: %d}\n",
			info.Origin.X, info.Origin.Y, info.Origin.W, info.Origin.H)
		fmt.Printf("  Size:     {W: %d, H: %d}\n", info.Size.W, info.Size.H)
		fmt.Printf("  Scale:    %.2f\n", info.ScaleFactor)
		fmt.Println("----------------------------------------")
	}

	if count == 0 {
		fmt.Println("No displays detected.")
		return
	}

	var minX, minY, maxX, maxY int
	for i, d := range displays {
		info := d.Info()
		x := info.Origin.X
		y := info.Origin.Y
		w := info.Origin.W
		h := info.Origin.H

		if i == 0 {
			minX, minY = x, y
			maxX, maxY = x+w, y+h
		} else {
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+w > maxX {
				maxX = x + w
			}
			if y+h > maxY {
				maxY = y + h
			}
		}
	}

	overviewWidth := maxX - minX
	overviewHeight := maxY - minY
	fmt.Printf("\nOverview canvas: %dx%d (offset: %d, %d)\n",
		overviewWidth, overviewHeight, minX, minY)

	axisMargin := 50
	canvasWidth := overviewWidth + axisMargin
	canvasHeight := overviewHeight + axisMargin

	overview := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	bgColor := color.RGBA{R: 40, G: 40, B: 40, A: 255}
	draw.Draw(overview, overview.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	drawAxes(overview, axisMargin, canvasWidth, canvasHeight, minX, minY, maxX, maxY)

	fmt.Println("\nCapturing displays...")
	for _, d := range displays {
		info := d.Info()
		idx := info.Index

		img, err := d.CaptureRect(0, 0, info.Size.W, info.Size.H, captureOptions)
		if err != nil {
			fmt.Printf("  Display #%d: capture failed: %v\n", idx, err)
			continue
		}

		filename := fmt.Sprintf("display_%d.png", idx)
		if err := savePNG(img, filename); err != nil {
			fmt.Printf("  Display #%d: save failed: %v\n", idx, err)
			continue
		}
		fmt.Printf("  Display #%d: saved to %s (%dx%d)\n",
			idx, filename, img.Bounds().Dx(), img.Bounds().Dy())

		posX := info.Origin.X - minX + axisMargin
		posY := info.Origin.Y - minY + axisMargin
		destW := info.Origin.W
		destH := info.Origin.H

		destRect := image.Rect(posX, posY, posX+destW, posY+destH)
		draw.CatmullRom.Scale(overview, destRect, img, img.Bounds(), draw.Over, nil)

		drawOriginLabel(overview, posX, posY, info.Origin.X, info.Origin.Y)
		drawDisplayInfo(overview, posX, posY, destW, destH, info, captureBackendLabel(captureOptions.Backend))
	}

	overviewFile := "display_overview.png"
	if err := savePNG(overview, overviewFile); err != nil {
		fmt.Printf("\nFailed to save overview: %v\n", err)
	} else {
		fmt.Printf("\nOverview saved to %s (%dx%d)\n",
			overviewFile, canvasWidth, canvasHeight)
	}

	fmt.Println("\n========================================")
	fmt.Println("Done!")
	fmt.Println("========================================")

	var logBuilder strings.Builder
	logBuilder.WriteString("DeskAct Capture Example\n")
	logBuilder.WriteString(fmt.Sprintf("Go version: %s\n", runtime.Version()))
	logBuilder.WriteString(fmt.Sprintf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	logBuilder.WriteString(fmt.Sprintf("DPI init selected: %v\n", dpiSelected))
	logBuilder.WriteString(fmt.Sprintf("DPI init result: %s\n", dpiResult))
	logBuilder.WriteString(fmt.Sprintf("Capture backend selected: %v\n", backendSelected))
	logBuilder.WriteString(fmt.Sprintf("Capture backend result: %s\n", captureBackendLabel(captureOptions.Backend)))
	logBuilder.WriteString(fmt.Sprintf("Excluded window IDs: %s\n", formatExcludedWindowIDs(captureOptions.ExcludedWindowIDs)))
	logBuilder.WriteString(fmt.Sprintf("Total displays: %d\n", count))
	for _, d := range displays {
		info := d.Info()
		logBuilder.WriteString(fmt.Sprintf("Display #%d: ID=%d, IsMain=%v, Origin=(%d,%d,%d,%d), Size=%dx%d, Scale=%.2f\n",
			info.Index, info.ID, info.IsMain, info.Origin.X, info.Origin.Y, info.Origin.W, info.Origin.H, info.Size.W, info.Size.H, info.ScaleFactor))
	}
	logBuilder.WriteString(fmt.Sprintf("Overview canvas: %dx%d\n", canvasWidth, canvasHeight))

	fmt.Println("\nPress 's' to save log and exit, or 'e' to exit directly:")
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "s" {
			if err := os.WriteFile("capture_log.txt", []byte(logBuilder.String()), 0644); err != nil {
				fmt.Printf("Failed to save log: %v\n", err)
			} else {
				fmt.Println("Log saved to capture_log.txt")
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

func promptCaptureBackend(defaultBackend deskact.CaptureBackend) (bool, deskact.CaptureBackend) {
	reader := bufio.NewReader(os.Stdin)
	options := availableCaptureBackends()
	if len(options) == 0 {
		return false, defaultBackend
	}

	fmt.Printf("\nSelect capture backend [Enter for %s]:\n", captureBackendLabel(defaultBackend))
	for _, option := range options {
		fmt.Printf("  %s) %s\n", option.key, captureBackendLabel(option.backend))
	}

	for {
		fmt.Print("Backend: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			return false, defaultBackend
		}
		for _, option := range options {
			if input == option.key {
				return true, option.backend
			}
		}
		fmt.Printf("Please press Enter for %s or choose one of: ", captureBackendLabel(defaultBackend))
		for i, option := range options {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(option.key)
		}
		fmt.Println()
	}
}

func availableCaptureBackends() []struct {
	key     string
	backend deskact.CaptureBackend
} {
	switch runtime.GOOS {
	case "windows":
		return []struct {
			key     string
			backend deskact.CaptureBackend
		}{
			{key: "d", backend: deskact.CaptureBackendDXGI},
			{key: "g", backend: deskact.CaptureBackendGDI},
		}
	case "darwin":
		return []struct {
			key     string
			backend deskact.CaptureBackend
		}{
			{key: "s", backend: deskact.CaptureBackendScreenCaptureKit},
			{key: "c", backend: deskact.CaptureBackendCGDisplay},
		}
	default:
		return []struct {
			key     string
			backend deskact.CaptureBackend
		}{}
	}
}

func captureBackendLabel(backend deskact.CaptureBackend) string {
	switch backend {
	case deskact.CaptureBackendDefault:
		return "default"
	case deskact.CaptureBackendDXGI:
		return "dxgi"
	case deskact.CaptureBackendGDI:
		return "gdi"
	case deskact.CaptureBackendScreenCaptureKit:
		return "screencapturekit"
	case deskact.CaptureBackendCGDisplay:
		return "cgdisplay"
	default:
		return string(backend)
	}
}

func promptExcludedWindowIDs(backend deskact.CaptureBackend) []uint64 {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nOptional: enter macOS excluded window IDs [Enter for none].\n")
		fmt.Printf("Current backend: %s. Window exclusion is supported by %s.\n", captureBackendLabel(backend), captureBackendLabel(deskact.CaptureBackendScreenCaptureKit))
		fmt.Print("Excluded window IDs: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return nil
		}

		fields := strings.FieldsFunc(input, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		})
		if len(fields) == 0 {
			return nil
		}

		ids := make([]uint64, 0, len(fields))
		seen := make(map[uint64]struct{}, len(fields))
		valid := true
		for _, field := range fields {
			id, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				fmt.Printf("Invalid window ID %q. Enter decimal uint64 IDs separated by commas or spaces.\n", field)
				valid = false
				break
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if valid {
			return ids
		}
	}
}

func formatExcludedWindowIDs(ids []uint64) string {
	if len(ids) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatUint(id, 10))
	}
	return strings.Join(parts, ", ")
}

func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func drawDisplayInfo(img *image.RGBA, displayX, displayY, displayW, displayH int, info deskact.DisplayInfo, backendLabel string) {
	lines := []string{
		fmt.Sprintf("Display #%d", info.Index),
		fmt.Sprintf("Backend: %s", backendLabel),
		fmt.Sprintf("Resolution: %dx%d", info.Size.W, info.Size.H),
		fmt.Sprintf("Origin: (%d, %d, %d, %d)", info.Origin.X, info.Origin.Y, info.Origin.W, info.Origin.H),
		fmt.Sprintf("Size: %dx%d", info.Size.W, info.Size.H),
		fmt.Sprintf("Scale: %.2f", info.ScaleFactor),
	}

	face := basicfont.Face7x13

	lineHeight := 16
	padding := 10
	maxWidth := 0

	for _, line := range lines {
		w := font.MeasureString(face, line).Ceil()
		if w > maxWidth {
			maxWidth = w
		}
	}

	boxWidth := maxWidth + padding*2
	boxHeight := len(lines)*lineHeight + padding*2

	boxX := displayX + displayW - boxWidth - padding
	boxY := displayY + displayH - boxHeight - padding

	for y := boxY; y < boxY+boxHeight; y++ {
		for x := boxX; x < boxX+boxWidth; x++ {
			if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 180})
			}
		}
	}

	textColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for i, line := range lines {
		x := boxX + padding
		y := boxY + padding + (i+1)*lineHeight - 3

		drawText(img, x, y, line, face, textColor)
	}
}

func drawText(img *image.RGBA, x, y int, text string, face font.Face, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}

func drawAxes(img *image.RGBA, margin, canvasW, canvasH, minX, minY, maxX, maxY int) {
	face := basicfont.Face7x13
	axisColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	tickColor := color.RGBA{R: 150, G: 150, B: 150, A: 255}

	for y := margin; y < canvasH; y++ {
		img.Set(margin-1, y, axisColor)
		img.Set(margin, y, axisColor)
	}

	for x := margin; x < canvasW; x++ {
		img.Set(x, margin-1, axisColor)
		img.Set(x, margin, axisColor)
	}

	contentW := maxX - minX
	tickInterval := calculateTickInterval(contentW)
	startTick := (minX / tickInterval) * tickInterval
	if startTick < minX {
		startTick += tickInterval
	}

	for tick := startTick; tick <= maxX; tick += tickInterval {
		x := tick - minX + margin
		for ty := margin - 5; ty < margin; ty++ {
			img.Set(x, ty, tickColor)
		}
		label := fmt.Sprintf("%d", tick)
		labelW := font.MeasureString(face, label).Ceil()
		drawText(img, x-labelW/2, margin-10, label, face, axisColor)
	}

	contentH := maxY - minY
	tickIntervalY := calculateTickInterval(contentH)
	startTickY := (minY / tickIntervalY) * tickIntervalY
	if startTickY < minY {
		startTickY += tickIntervalY
	}

	for tick := startTickY; tick <= maxY; tick += tickIntervalY {
		y := tick - minY + margin
		for tx := margin - 5; tx < margin; tx++ {
			img.Set(tx, y, tickColor)
		}
		label := fmt.Sprintf("%d", tick)
		labelW := font.MeasureString(face, label).Ceil()
		drawText(img, margin-labelW-8, y+4, label, face, axisColor)
	}

	originLabel := fmt.Sprintf("(%d,%d)", minX, minY)
	drawText(img, 2, margin-10, originLabel, face, axisColor)
}

func calculateTickInterval(size int) int {
	if size <= 500 {
		return 100
	}
	if size <= 1000 {
		return 200
	}
	if size <= 2000 {
		return 500
	}
	if size <= 5000 {
		return 1000
	}
	return 2000
}

func drawOriginLabel(img *image.RGBA, x, y, originX, originY int) {
	face := basicfont.Face7x13
	label := fmt.Sprintf("(%d, %d)", originX, originY)

	padding := 5
	labelW := font.MeasureString(face, label).Ceil()
	boxW := labelW + padding*2
	boxH := 16 + padding

	for py := y; py < y+boxH; py++ {
		for px := x; px < x+boxW; px++ {
			if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
				img.Set(px, py, color.RGBA{R: 0, G: 0, B: 0, A: 180})
			}
		}
	}

	textColor := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	drawText(img, x+padding, y+14, label, face, textColor)
}
