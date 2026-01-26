package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"runtime"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	deskact "github.com/PekingSpades/DeskAct"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Window Overlay Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")

	dpiSelected, dpiResult := promptDPIInit()
	fmt.Printf("DPI init selected: %v\n", dpiSelected)
	fmt.Printf("DPI init result: %s\n", dpiResult)

	displayOptions := deskact.DefaultDisplayOptions()
	windowOptions := deskact.DefaultWindowOptions()
	windowOptions.DPIAware = displayOptions.DPIAware

	displays := deskact.AllDisplays(displayOptions)
	if len(displays) == 0 {
		fmt.Println("No displays detected.")
		return
	}

	windows, err := deskact.ListWindows(windowOptions)
	if err != nil {
		if errors.Is(err, deskact.ErrWindowUnsupported) {
			fmt.Println("Window listing is not supported on this platform.")
			return
		}
		fmt.Printf("ListWindows error: %v\n", err)
		return
	}
	fmt.Printf("Detected %d windows\n", len(windows))

	face, closeFace, faceInfo := loadFontFace()
	defer closeFace()
	if faceInfo != "" {
		fmt.Printf("Using font: %s\n", faceInfo)
	} else {
		fmt.Println("Using built-in ASCII font (CJK may not render).")
	}

	minX, minY, maxX, maxY := displayBounds(displays)
	overviewWidth := maxX - minX
	overviewHeight := maxY - minY
	if overviewWidth <= 0 || overviewHeight <= 0 {
		fmt.Println("Invalid overview bounds.")
		return
	}

	axisMargin := 50
	canvasWidth := overviewWidth + axisMargin
	canvasHeight := overviewHeight + axisMargin
	overview := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	bgColor := color.RGBA{R: 40, G: 40, B: 40, A: 255}
	draw.Draw(overview, overview.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	drawAxes(overview, axisMargin, canvasWidth, canvasHeight, minX, minY, maxX, maxY)

	fmt.Println("\nCapturing displays...")
	for _, d := range displays {
		info := d.Info()
		if info.Size.W <= 0 || info.Size.H <= 0 {
			continue
		}

		img, err := d.CaptureRect(0, 0, info.Size.W, info.Size.H, deskact.DefaultCaptureOptions())
		if err != nil {
			fmt.Printf("  Display #%d: capture failed: %v\n", info.Index, err)
			continue
		}

		annotateDisplay(img, windows, info.Index, face)
		displayFile := fmt.Sprintf("window_display_%d.png", info.Index)
		if err := savePNG(displayFile, img); err != nil {
			fmt.Printf("  Display #%d: save failed: %v\n", info.Index, err)
		} else {
			fmt.Printf("  Display #%d: saved to %s (%dx%d)\n",
				info.Index, displayFile, img.Bounds().Dx(), img.Bounds().Dy())
		}

		posX := info.Origin.X - minX + axisMargin
		posY := info.Origin.Y - minY + axisMargin
		destW := info.Origin.W
		destH := info.Origin.H
		if destW <= 0 || destH <= 0 {
			continue
		}

		destRect := image.Rect(posX, posY, posX+destW, posY+destH)
		draw.CatmullRom.Scale(overview, destRect, img, img.Bounds(), draw.Over, nil)
	}

	fmt.Println("\nOverlaying windows...")
	overlayWindows(overview, windows, displays, minX, minY, axisMargin, face)

	overviewFile := "window_overview.png"
	if err := savePNG(overviewFile, overview); err != nil {
		fmt.Printf("\nFailed to save overview: %v\n", err)
	} else {
		fmt.Printf("\nOverview saved to %s (%dx%d)\n", overviewFile, canvasWidth, canvasHeight)
	}
}

func displayBounds(displays []*deskact.Display) (minX, minY, maxX, maxY int) {
	for i, d := range displays {
		info := d.Info()
		x := info.Origin.X
		y := info.Origin.Y
		w := info.Origin.W
		h := info.Origin.H

		if i == 0 {
			minX, minY = x, y
			maxX, maxY = x+w, y+h
			continue
		}
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
	return minX, minY, maxX, maxY
}

func overlayWindows(img *image.RGBA, windows []deskact.WindowInfo, displays []*deskact.Display, minX, minY, margin int, face font.Face) {
	if img == nil {
		return
	}
	red := color.RGBA{R: 220, G: 40, B: 40, A: 255}
	displayByIndex := make(map[int]deskact.DisplayInfo, len(displays))
	for _, d := range displays {
		displayByIndex[d.Index()] = d.Info()
	}

	for _, w := range windows {
		title := strings.TrimSpace(w.Title)
		if title == "" {
			title = fmt.Sprintf("PID %d", w.PID)
		}
		title = truncateLabel(title, 60)

		labeled := false
		for _, region := range w.DisplayRegions {
			info, ok := displayByIndex[region.DisplayIndex]
			if !ok {
				continue
			}
			virt, ok := physicalToVirtualRect(region.PhysicalRect, info)
			if !ok {
				continue
			}
			x := virt.X - minX + margin
			y := virt.Y - minY + margin
			rect := image.Rect(x, y, x+virt.W, y+virt.H)
			drawRect(img, rect, red, 2)
			if !labeled {
				drawLabel(img, rect.Min.X+2, rect.Min.Y+2, title, red, face)
				labeled = true
			}
		}
		if labeled {
			continue
		}
		if w.Bounds.W <= 0 || w.Bounds.H <= 0 {
			continue
		}
		x := w.Bounds.X - minX + margin
		y := w.Bounds.Y - minY + margin
		rect := image.Rect(x, y, x+w.Bounds.W, y+w.Bounds.H)
		drawRect(img, rect, red, 2)
		drawLabel(img, rect.Min.X+2, rect.Min.Y+2, title, red, face)
	}
}

func physicalToVirtualRect(phys deskact.Rect, info deskact.DisplayInfo) (deskact.Rect, bool) {
	if phys.W <= 0 || phys.H <= 0 {
		return deskact.Rect{}, false
	}
	if info.Size.W <= 0 || info.Size.H <= 0 || info.Origin.W <= 0 || info.Origin.H <= 0 {
		return deskact.Rect{}, false
	}

	scaleX := float64(info.Origin.W) / float64(info.Size.W)
	scaleY := float64(info.Origin.H) / float64(info.Size.H)
	x0 := int(math.Round(float64(phys.X) * scaleX))
	y0 := int(math.Round(float64(phys.Y) * scaleY))
	x1 := int(math.Round(float64(phys.X+phys.W) * scaleX))
	y1 := int(math.Round(float64(phys.Y+phys.H) * scaleY))
	w := x1 - x0
	h := y1 - y0
	if w <= 0 || h <= 0 {
		return deskact.Rect{}, false
	}
	return deskact.Rect{
		Point: deskact.Point{X: info.Origin.X + x0, Y: info.Origin.Y + y0},
		Size:  deskact.Size{W: w, H: h},
	}, true
}

func annotateDisplay(img *image.RGBA, windows []deskact.WindowInfo, displayIndex int, face font.Face) {
	if img == nil {
		return
	}
	red := color.RGBA{R: 220, G: 40, B: 40, A: 255}

	for _, w := range windows {
		title := strings.TrimSpace(w.Title)
		if title == "" {
			title = fmt.Sprintf("PID %d", w.PID)
		}
		title = truncateLabel(title, 60)

		for _, region := range w.DisplayRegions {
			if region.DisplayIndex != displayIndex {
				continue
			}
			if region.PhysicalRect.W <= 0 || region.PhysicalRect.H <= 0 {
				continue
			}
			x := region.PhysicalRect.X
			y := region.PhysicalRect.Y
			rect := image.Rect(x, y, x+region.PhysicalRect.W, y+region.PhysicalRect.H)
			drawRect(img, rect, red, 2)
			drawLabel(img, rect.Min.X+2, rect.Min.Y+2, title, red, face)
		}
	}
}

func loadFontFace() (font.Face, func(), string) {
	size := 14.0
	paths := fontPaths()
	for _, path := range paths {
		face, err := loadFontFromPath(path, size)
		if err == nil {
			return face, func() { closeFace(face) }, path
		}
	}
	return basicfont.Face7x13, func() {}, ""
}

func fontPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			`C:\Windows\Fonts\msyh.ttc`,
			`C:\Windows\Fonts\simsun.ttc`,
			`C:\Windows\Fonts\simhei.ttf`,
			`C:\Windows\Fonts\msyh.ttf`,
			`C:\Windows\Fonts\segoeui.ttf`,
		}
	case "darwin":
		return []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/System/Library/Fonts/AppleGothic.ttf",
		}
	default:
		return []string{
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJKSC-Regular.otf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		}
	}
}

func loadFontFromPath(path string, size float64) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	collection, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, err
	}
	for i := 0; i < collection.NumFonts(); i++ {
		f, err := collection.Font(i)
		if err != nil {
			continue
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err == nil {
			return face, nil
		}
	}
	return nil, errors.New("no usable font in collection")
}

func closeFace(face font.Face) {
	if face == nil {
		return
	}
	if closer, ok := face.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func truncateLabel(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func drawRect(img *image.RGBA, rect image.Rectangle, c color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	b := img.Bounds()
	rect = rect.Intersect(b)
	if rect.Empty() {
		return
	}

	for t := 0; t < thickness; t++ {
		x0 := rect.Min.X + t
		y0 := rect.Min.Y + t
		x1 := rect.Max.X - 1 - t
		y1 := rect.Max.Y - 1 - t
		if x1 < x0 || y1 < y0 {
			break
		}
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y0, c)
			img.SetRGBA(x, y1, c)
		}
		for y := y0; y <= y1; y++ {
			img.SetRGBA(x0, y, c)
			img.SetRGBA(x1, y, c)
		}
	}
}

func drawLabel(img *image.RGBA, x, y int, text string, c color.RGBA, face font.Face) {
	if text == "" || img == nil || face == nil {
		return
	}
	bounds := img.Bounds()
	if x < bounds.Min.X {
		x = bounds.Min.X
	}
	if y < bounds.Min.Y {
		y = bounds.Min.Y
	}
	ascent := face.Metrics().Ascent.Round()
	descent := face.Metrics().Descent.Round()
	baseline := y + ascent
	if baseline+descent > bounds.Max.Y {
		baseline = bounds.Max.Y - descent
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(text)
}

func savePNG(filename string, img image.Image) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
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

func drawText(img *image.RGBA, x, y int, text string, face font.Face, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}
