package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	deskact "github.com/PekingSpades/DeskAct"
	"golang.org/x/image/draw"
)

const (
	iconSize     = 32
	saveIconSize = 64
	appsPerRow   = 5

	appsRootDirName  = ".apps"
	installedDirName = ".installed"
	desktopDirName   = ".desktop"
	iconsDirName     = "icons"
	markdownFileName = "README.md"
	pdfFileName      = "README.pdf"
	logFileName      = "output.log"

	pdfPageWidth  = 612.0
	pdfPageHeight = 792.0
	pdfMargin     = 54.0
	pdfFontSize   = 12.0
	pdfLeading    = 16.0
	pdfIconSize   = 48.0
	pdfTextGap    = 12.0
	pdfSectionGap = 18.0
	pdfEntryGap   = 12.0
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DeskAct Apps Example")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("========================================")
	defer waitForExit()

	listMode := promptListMode()
	appsDir := resolveAppsDir()
	outputMode := promptOutputMode(appsDir)

	groups := loadAppGroups(listMode)
	if len(groups) == 0 {
		fmt.Println("No app groups selected.")
		return
	}

	for _, group := range groups {
		missing, other := splitAppErrors(group.Err)
		if other != nil {
			fmt.Printf("Warning (%s): %v\n", group.Label, other)
		}
		if missing > 0 && outputMode == "console" {
			fmt.Printf("Missing icons (%s): %d\n", group.Label, missing)
		}

		switch outputMode {
		case "console":
			printGroup(group)
		case "save":
			saveGroup(group, appsDir)
		}
	}
}

func waitForExit() {
	fmt.Print("\n按任意键退出程序...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadByte()
}

type appGroup struct {
	Mode  string
	Label string
	Apps  []deskact.AppInfo
	Err   error
}

func loadAppGroups(listMode string) []appGroup {
	var groups []appGroup
	if listMode == "installed" || listMode == "both" {
		apps, err := deskact.InstalledApps()
		groups = append(groups, appGroup{
			Mode:  "installed",
			Label: "Installed apps",
			Apps:  apps,
			Err:   err,
		})
	}
	if listMode == "desktop" || listMode == "both" {
		apps, err := deskact.DesktopApps()
		groups = append(groups, appGroup{
			Mode:  "desktop",
			Label: "Desktop apps",
			Apps:  apps,
			Err:   err,
		})
	}
	return groups
}

func printGroup(group appGroup) {
	fmt.Printf("\n%s\n", group.Label)
	fmt.Printf("Total apps: %d\n", len(group.Apps))
	fmt.Println("----------------------------------------")
	printApps(group.Apps)
}

func saveGroup(group appGroup, appsDir string) {
	groupDir := filepath.Join(appsDir, dirNameForMode(group.Mode))
	iconsDir := filepath.Join(groupDir, iconsDirName)

	fmt.Printf("\n%s\n", group.Label)
	fmt.Printf("Total apps: %d\n", len(group.Apps))

	apps, missing, saveErr := saveIcons(group.Apps, iconsDir, iconsDirName)
	savedCount := countSavedIcons(apps)
	if saveErr != nil {
		fmt.Printf("Save errors (%s): %v\n", group.Label, saveErr)
	}
	if missing > 0 {
		fmt.Printf("Missing icons (%s): %d\n", group.Label, missing)
	}
	mdErr := writeMarkdown(groupDir, apps, group.Mode, len(group.Apps), missing, saveErr)
	if mdErr != nil {
		fmt.Printf("Markdown errors (%s): %v\n", group.Label, mdErr)
	}
	pdfErr := writePDF(groupDir, apps, group.Mode, len(group.Apps), missing, saveErr)
	if pdfErr != nil {
		fmt.Printf("PDF errors (%s): %v\n", group.Label, pdfErr)
	}
	if logErr := writeLog(groupDir, group, apps, missing, saveErr, mdErr, pdfErr); logErr != nil {
		fmt.Printf("Log errors (%s): %v\n", group.Label, logErr)
	}
	fmt.Printf("Saved %d icons to %s\n", savedCount, iconsDir)
}

func dirNameForMode(mode string) string {
	switch mode {
	case "installed":
		return installedDirName
	case "desktop":
		return desktopDirName
	default:
		return mode
	}
}

func promptListMode() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Select app list:")
		fmt.Println("1) Installed apps")
		fmt.Println("2) Desktop apps")
		fmt.Println("3) Both installed + desktop apps")
		fmt.Print("Choice [1/2/3]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "1", "installed", "i":
			return "installed"
		case "2", "desktop", "d":
			return "desktop"
		case "3", "both", "b", "all":
			return "both"
		default:
			fmt.Println("Please enter 1, 2, or 3.")
		}
	}
}

func promptOutputMode(appsDir string) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nSelect output mode:")
		fmt.Println("1) Print to console")
		fmt.Printf("2) Save to .apps directory (%s)\n", appsDir)
		fmt.Print("Choice [1/2]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "1", "console", "c":
			return "console"
		case "2", "save", "s":
			return "save"
		default:
			fmt.Println("Please enter 1 or 2.")
		}
	}
}

func resolveAppsDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Join(".", appsRootDirName)
	}
	return filepath.Join(wd, appsRootDirName)
}

func printApps(apps []deskact.AppInfo) {
	for i, app := range apps {
		fmt.Printf("\n[%d] %s\n", i+1, app.Name)
		fmt.Printf("Path: %s\n", app.Path)
		if app.Icon == nil {
			fmt.Println("Icon: (none)")
			continue
		}
		fmt.Printf("Icon: %dx%d\n", app.Icon.Bounds().Dx(), app.Icon.Bounds().Dy())
		printIconDots(app.Icon)
	}
}

func printIconDots(icon *image.RGBA) {
	scaled := resizeIcon(icon, iconSize, draw.NearestNeighbor)
	if scaled == nil {
		fmt.Println("(icon size invalid)")
		return
	}

	for y := 0; y < iconSize; y++ {
		row := scaled.Pix[y*scaled.Stride:]
		var sb strings.Builder
		sb.Grow(iconSize * 2)
		for x := 0; x < iconSize; x++ {
			if row[x*4+3] != 0 {
				sb.WriteByte('.')
				sb.WriteByte(' ')
			} else {
				sb.WriteByte(' ')
				sb.WriteByte(' ')
			}
		}
		fmt.Println(sb.String())
	}
}

func resizeIcon(src *image.RGBA, size int, scaler draw.Scaler) *image.RGBA {
	if src == nil || size <= 0 {
		return nil
	}
	if src.Bounds().Dx() <= 0 || src.Bounds().Dy() <= 0 {
		return nil
	}
	if src.Bounds().Dx() == size && src.Bounds().Dy() == size {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	scaler.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

type appEntry struct {
	Name         string
	Path         string
	IconPath     string
	IconFilePath string
	HasIcon      bool
}

func saveIcons(apps []deskact.AppInfo, iconsDir string, iconRelDir string) ([]appEntry, int, error) {
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		return nil, 0, err
	}

	entries := make([]appEntry, 0, len(apps))
	missing := 0
	var errs []error
	for i, app := range apps {
		entry := appEntry{
			Name: app.Name,
			Path: app.Path,
		}
		if app.Icon == nil {
			missing++
			entries = append(entries, entry)
			continue
		}
		scaled := resizeIcon(app.Icon, saveIconSize, draw.CatmullRom)
		if scaled == nil {
			missing++
			entries = append(entries, entry)
			continue
		}
		name := sanitizeFileName(app.Name)
		if name == "" {
			name = "app"
		}
		fileBase := name + "_" + strconv.Itoa(i+1)
		path := filepath.Join(iconsDir, fileBase+".png")

		path = ensureUniquePath(path)
		if err := savePNG(scaled, path); err != nil {
			if errors.Is(err, deskact.ErrIconNotFound) {
				missing++
			} else {
				errs = append(errs, err)
			}
			entries = append(entries, entry)
			continue
		}
		entry.HasIcon = true
		entry.IconFilePath = path
		entry.IconPath = buildIconRelPath(iconRelDir, filepath.Base(path))
		entries = append(entries, entry)
	}

	if len(errs) > 0 {
		return entries, missing, errors.Join(errs...)
	}
	return entries, missing, nil
}

func buildIconRelPath(iconRelDir string, fileName string) string {
	if iconRelDir == "" {
		return filepath.ToSlash(fileName)
	}
	return filepath.ToSlash(filepath.Join(iconRelDir, fileName))
}

func filterAppsWithIcons(apps []appEntry) []appEntry {
	if len(apps) == 0 {
		return nil
	}
	filtered := make([]appEntry, 0, len(apps))
	for _, app := range apps {
		if app.HasIcon {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

func countSavedIcons(apps []appEntry) int {
	count := 0
	for _, app := range apps {
		if app.HasIcon {
			count++
		}
	}
	return count
}

func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func sanitizeFileName(name string) string {
	var sb strings.Builder
	sb.Grow(len(name))
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	return strings.Trim(sb.String(), "_")
}

func ensureUniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	ext := filepath.Ext(path)
	for i := 2; ; i++ {
		candidate := filepath.Join(dir, base+"_"+strconv.Itoa(i)+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func writeMarkdown(dir string, apps []appEntry, listMode string, totalApps int, missing int, saveErr error) error {
	iconsOnly := filterAppsWithIcons(apps)
	savedCount := len(iconsOnly)
	var sb strings.Builder
	sb.WriteString("# DeskAct Apps\n\n")
	writeTable(&sb, iconsOnly)
	sb.WriteString("\n---\n\n")
	sb.WriteString("Summary\n\n")
	sb.WriteString(fmt.Sprintf("- List mode: %s\n", listMode))
	sb.WriteString(fmt.Sprintf("- Total apps: %d\n", totalApps))
	sb.WriteString(fmt.Sprintf("- Saved icons: %d\n", savedCount))
	sb.WriteString(fmt.Sprintf("- Missing icons: %d\n", missing))
	sb.WriteString(fmt.Sprintf("- Generated at: %s\n", time.Now().Format(time.RFC3339)))
	if saveErr != nil {
		sb.WriteString(fmt.Sprintf("- Save errors: %v\n", saveErr))
	}

	path := filepath.Join(dir, markdownFileName)
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writePDF(dir string, apps []appEntry, listMode string, totalApps int, missing int, saveErr error) error {
	savedCount := countSavedIcons(apps)
	summary := buildPDFSummary(listMode, totalApps, savedCount, missing, saveErr)
	data, buildErr := buildPDF(summary, apps)
	if len(data) == 0 {
		if buildErr != nil {
			return buildErr
		}
		return errors.New("pdf generation returned empty data")
	}
	path := filepath.Join(dir, pdfFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return buildErr
}

func buildPDFSummary(listMode string, totalApps int, savedCount int, missing int, saveErr error) []string {
	lines := []string{
		"DeskAct Apps",
		fmt.Sprintf("List mode: %s", listMode),
		fmt.Sprintf("Total apps: %d", totalApps),
		fmt.Sprintf("Saved icons: %d", savedCount),
		fmt.Sprintf("Missing icons: %d", missing),
		fmt.Sprintf("Generated at: %s", time.Now().Format(time.RFC3339)),
	}
	if saveErr != nil {
		lines = append(lines, fmt.Sprintf("Save errors: %v", saveErr))
	}
	return lines
}

type pdfImage struct {
	Name     string
	Width    int
	Height   int
	Data     []byte
	ObjectID int
}

type pdfPage struct {
	Content string
	Images  map[string]*pdfImage
}

type pdfDocument struct {
	Pages  []pdfPage
	Images []*pdfImage
}

func buildPDF(summary []string, apps []appEntry) ([]byte, error) {
	doc, err := buildPDFDocument(summary, apps)
	if doc == nil {
		return nil, err
	}
	data := renderPDF(doc)
	return data, err
}

func buildPDFDocument(summary []string, apps []appEntry) (*pdfDocument, error) {
	doc := &pdfDocument{}
	imageRegistry := make(map[string]*pdfImage)
	var imageErrors []error

	page := pdfPage{Images: make(map[string]*pdfImage)}
	var content strings.Builder
	y := pdfPageHeight - pdfMargin

	summaryWidth := pdfPageWidth - 2*pdfMargin
	summaryMaxRunes := maxRunesForWidth(summaryWidth, pdfFontSize)
	summaryLines := wrapLines(summary, summaryMaxRunes)
	summaryHeight := textBlockHeight(summaryLines, pdfLeading, pdfFontSize)
	addTextBlock(&content, summaryLines, pdfMargin, y-pdfFontSize, pdfFontSize, pdfLeading)
	y -= summaryHeight + pdfSectionGap

	textX := pdfMargin + pdfIconSize + pdfTextGap
	textWidth := pdfPageWidth - pdfMargin - textX
	textMaxRunes := maxRunesForWidth(textWidth, pdfFontSize)

	for _, app := range apps {
		iconStatus := "图标: (无)"
		var iconName string
		var iconImage *pdfImage
		if app.HasIcon && app.IconFilePath != "" {
			iconStatus = fmt.Sprintf("图标: %s", app.IconPath)
			img, err := getPDFImage(app.IconFilePath, imageRegistry, &doc.Images)
			if err != nil {
				imageErrors = append(imageErrors, err)
				iconStatus = "图标: (加载失败)"
			} else {
				iconName = img.Name
				iconImage = img
			}
		}

		lines := buildAppLines(app, iconStatus, textMaxRunes)
		entryHeight := maxFloat(pdfIconSize, textBlockHeight(lines, pdfLeading, pdfFontSize)) + pdfEntryGap
		if y-entryHeight < pdfMargin {
			page.Content = content.String()
			doc.Pages = append(doc.Pages, page)
			page = pdfPage{Images: make(map[string]*pdfImage)}
			content.Reset()
			y = pdfPageHeight - pdfMargin
		}

		if iconImage != nil {
			page.Images[iconImage.Name] = iconImage
		}
		if iconName != "" {
			addImageCommand(&content, iconName, pdfMargin, y-pdfIconSize, pdfIconSize, pdfIconSize)
		}
		addTextBlock(&content, lines, textX, y-pdfFontSize, pdfFontSize, pdfLeading)
		y -= entryHeight
	}

	page.Content = content.String()
	doc.Pages = append(doc.Pages, page)
	return doc, errors.Join(imageErrors...)
}

func renderPDF(doc *pdfDocument) []byte {
	pageCount := len(doc.Pages)
	imageCount := len(doc.Images)
	if pageCount == 0 {
		return nil
	}

	catalogID := 1
	pagesID := 2
	firstPageID := 3
	firstContentID := firstPageID + pageCount
	firstImageID := firstContentID + pageCount
	fontType0ID := firstImageID + imageCount
	cidFontID := fontType0ID + 1
	fontDescriptorID := fontType0ID + 2
	helveticaID := fontType0ID + 3
	lastID := helveticaID

	for i, img := range doc.Images {
		img.ObjectID = firstImageID + i
	}

	objects := make(map[int][]byte, lastID+1)
	objects[catalogID] = []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID))

	kids := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		kids[i] = fmt.Sprintf("%d 0 R", firstPageID+i)
	}
	objects[pagesID] = []byte(fmt.Sprintf("<< /Type /Pages /Kids [ %s ] /Count %d >>", strings.Join(kids, " "), pageCount))

	for i, page := range doc.Pages {
		pageID := firstPageID + i
		contentID := firstContentID + i
		objects[contentID] = buildStreamObject([]byte(page.Content))

		resources := buildPDFResources(page.Images, fontType0ID, helveticaID)
		pageObj := fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.0f %.0f] /Contents %d 0 R /Resources %s >>",
			pagesID, pdfPageWidth, pdfPageHeight, contentID, resources)
		objects[pageID] = []byte(pageObj)
	}

	for _, img := range doc.Images {
		objects[img.ObjectID] = buildImageObject(img)
	}

	// Built-in CJK font so Unicode app names render without external font files.
	objects[fontType0ID] = []byte(fmt.Sprintf("<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [ %d 0 R ] >>", cidFontID))
	objects[cidFontID] = []byte(fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> /FontDescriptor %d 0 R /DW 1000 >>", fontDescriptorID))
	objects[fontDescriptorID] = []byte("<< /Type /FontDescriptor /FontName /STSong-Light /Flags 4 /FontBBox [0 -260 1000 880] /ItalicAngle 0 /Ascent 880 /Descent -260 /CapHeight 750 /StemV 80 >>")
	objects[helveticaID] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, lastID+1)
	for id := 1; id <= lastID; id++ {
		content := objects[id]
		offsets[id] = buf.Len()
		buf.WriteString(fmt.Sprintf("%d 0 obj\n", id))
		buf.Write(content)
		if len(content) == 0 || content[len(content)-1] != '\n' {
			buf.WriteByte('\n')
		}
		buf.WriteString("endobj\n")
	}

	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", lastID+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= lastID; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root %d 0 R >>\n", lastID+1, catalogID))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")
	return buf.Bytes()
}

func buildPDFResources(images map[string]*pdfImage, cjkFontID int, latinFontID int) string {
	var sb strings.Builder
	sb.WriteString("<< /Font << /F1 ")
	sb.WriteString(strconv.Itoa(cjkFontID))
	sb.WriteString(" 0 R /F2 ")
	sb.WriteString(strconv.Itoa(latinFontID))
	sb.WriteString(" 0 R >>")
	if len(images) > 0 {
		sb.WriteString(" /XObject << ")
		names := make([]string, 0, len(images))
		for name := range images {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			img := images[name]
			sb.WriteString("/")
			sb.WriteString(name)
			sb.WriteString(" ")
			sb.WriteString(strconv.Itoa(img.ObjectID))
			sb.WriteString(" 0 R ")
		}
		sb.WriteString(">>")
	}
	sb.WriteString(" >>")
	return sb.String()
}

func buildStreamObject(data []byte) []byte {
	var sb bytes.Buffer
	sb.WriteString(fmt.Sprintf("<< /Length %d >>\nstream\n", len(data)))
	sb.Write(data)
	sb.WriteString("\nendstream")
	return sb.Bytes()
}

func buildImageObject(img *pdfImage) []byte {
	var sb bytes.Buffer
	sb.WriteString(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n",
		img.Width, img.Height, len(img.Data)))
	sb.Write(img.Data)
	sb.WriteString("\nendstream")
	return sb.Bytes()
}

func addTextBlock(sb *strings.Builder, lines []string, x float64, y float64, fontSize float64, leading float64) {
	if len(lines) == 0 {
		return
	}
	sb.WriteString("BT\n")
	sb.WriteString(fmt.Sprintf("%.2f %.2f Td\n", x, y))
	sb.WriteString(fmt.Sprintf("%.2f TL\n", leading))
	lastFont := ""
	for i, line := range lines {
		segments := splitTextSegments(line)
		if len(segments) == 0 {
			if lastFont == "" {
				sb.WriteString(fmt.Sprintf("/F2 %.2f Tf\n", fontSize))
				lastFont = "F2"
			}
			sb.WriteString("() Tj\n")
		} else {
			for _, segment := range segments {
				if segment.Font != lastFont {
					sb.WriteString(fmt.Sprintf("/%s %.2f Tf\n", segment.Font, fontSize))
					lastFont = segment.Font
				}
				sb.WriteString(segment.PDFText)
				sb.WriteString(" Tj\n")
			}
		}
		if i != len(lines)-1 {
			sb.WriteString("T*\n")
		}
	}
	sb.WriteString("ET\n")
}

func addImageCommand(sb *strings.Builder, name string, x float64, y float64, w float64, h float64) {
	sb.WriteString("q\n")
	sb.WriteString(fmt.Sprintf("%.2f 0 0 %.2f %.2f %.2f cm\n", w, h, x, y))
	sb.WriteString("/")
	sb.WriteString(name)
	sb.WriteString(" Do\n")
	sb.WriteString("Q\n")
}

func textBlockHeight(lines []string, leading float64, fontSize float64) float64 {
	if len(lines) == 0 {
		return 0
	}
	return leading*float64(len(lines)-1) + fontSize
}

func maxRunesForWidth(width float64, fontSize float64) int {
	if fontSize <= 0 || width <= 0 {
		return 10
	}
	max := int(width / fontSize)
	if max < 10 {
		return 10
	}
	return max
}

func wrapLines(lines []string, maxRunes int) []string {
	var out []string
	for _, line := range lines {
		out = append(out, wrapText(line, maxRunes)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapText(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		return []string{text}
	}
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")

	var lines []string
	var sb strings.Builder
	count := 0
	for _, r := range text {
		sb.WriteRune(r)
		count++
		if count >= maxRunes {
			lines = append(lines, sb.String())
			sb.Reset()
			count = 0
		}
	}
	if sb.Len() > 0 {
		lines = append(lines, sb.String())
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func buildAppLines(app appEntry, iconStatus string, maxRunes int) []string {
	name := strings.TrimSpace(app.Name)
	if name == "" {
		name = "(unknown)"
	}
	path := strings.TrimSpace(app.Path)
	if path == "" {
		path = "(unknown)"
	}
	if iconStatus == "" {
		iconStatus = "图标: (无)"
	}

	var lines []string
	lines = append(lines, wrapText(fmt.Sprintf("应用名称: %s", name), maxRunes)...)
	lines = append(lines, wrapText(fmt.Sprintf("路径: %s", path), maxRunes)...)
	lines = append(lines, wrapText(iconStatus, maxRunes)...)
	return lines
}

type textSegment struct {
	Font    string
	PDFText string
}

func splitTextSegments(text string) []textSegment {
	if text == "" {
		return nil
	}
	var segments []textSegment
	var sb strings.Builder
	currentASCII := true
	currentSet := false

	flush := func() {
		if sb.Len() == 0 {
			return
		}
		if currentASCII {
			segments = append(segments, textSegment{
				Font:    "F2",
				PDFText: pdfLiteralText(sb.String()),
			})
		} else {
			segments = append(segments, textSegment{
				Font:    "F1",
				PDFText: pdfHexText(sb.String()),
			})
		}
		sb.Reset()
	}

	for _, r := range text {
		isASCII := r >= 32 && r <= 126
		if r < 32 || r == 127 {
			r = '?'
			isASCII = true
		}
		if !currentSet {
			currentASCII = isASCII
			currentSet = true
		}
		if isASCII != currentASCII {
			flush()
			currentASCII = isASCII
		}
		sb.WriteRune(r)
	}
	flush()
	return segments
}

func pdfHexText(text string) string {
	encoded := utf16.Encode([]rune(text))
	var sb strings.Builder
	sb.WriteString("<")
	for _, v := range encoded {
		sb.WriteString(fmt.Sprintf("%04X", v))
	}
	sb.WriteString(">")
	return sb.String()
}

func pdfLiteralText(text string) string {
	return "(" + pdfEscapeLiteral(text) + ")"
}

func pdfEscapeLiteral(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "\r", "\\r")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

func getPDFImage(path string, registry map[string]*pdfImage, images *[]*pdfImage) (*pdfImage, error) {
	if img, ok := registry[path]; ok {
		return img, nil
	}
	width, height, data, err := loadPDFImageData(path)
	if err != nil {
		return nil, err
	}
	img := &pdfImage{
		Name:   fmt.Sprintf("Im%d", len(*images)+1),
		Width:  width,
		Height: height,
		Data:   data,
	}
	registry[path] = img
	*images = append(*images, img)
	return img, nil
}

func loadPDFImageData(path string) (int, int, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return 0, 0, nil, err
	}
	return encodePDFImage(img)
}

func encodePDFImage(img image.Image) (int, int, []byte, error) {
	rgba := flattenToRGBA(img)
	width := rgba.Bounds().Dx()
	height := rgba.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return 0, 0, nil, errors.New("invalid image size")
	}

	raw := make([]byte, width*height*3)
	index := 0
	for y := 0; y < height; y++ {
		row := rgba.Pix[y*rgba.Stride:]
		for x := 0; x < width; x++ {
			p := x * 4
			raw[index] = row[p]
			raw[index+1] = row[p+1]
			raw[index+2] = row[p+2]
			index += 3
		}
	}

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return 0, 0, nil, err
	}
	if err := zw.Close(); err != nil {
		return 0, 0, nil, err
	}
	return width, height, buf.Bytes(), nil
}

func flattenToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	imagedraw.Draw(rgba, bounds, &image.Uniform{C: color.White}, image.Point{}, imagedraw.Src)
	imagedraw.Draw(rgba, bounds, img, bounds.Min, imagedraw.Over)
	return rgba
}

func maxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func writeLog(dir string, group appGroup, apps []appEntry, missing int, saveErr error, mdErr error, pdfErr error) error {
	var sb strings.Builder
	sb.WriteString("DeskAct Apps Log\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Group: %s\n", group.Label))
	sb.WriteString(fmt.Sprintf("List mode: %s\n", group.Mode))
	sb.WriteString(fmt.Sprintf("Total apps: %d\n", len(apps)))
	sb.WriteString(fmt.Sprintf("Saved icons: %d\n", countSavedIcons(apps)))
	sb.WriteString(fmt.Sprintf("Missing icons: %d\n", missing))
	if group.Err != nil {
		sb.WriteString(fmt.Sprintf("List errors: %v\n", group.Err))
	}
	if saveErr != nil {
		sb.WriteString(fmt.Sprintf("Save errors: %v\n", saveErr))
	}
	if mdErr != nil {
		sb.WriteString(fmt.Sprintf("Markdown errors: %v\n", mdErr))
	}
	if pdfErr != nil {
		sb.WriteString(fmt.Sprintf("PDF errors: %v\n", pdfErr))
	}
	path := filepath.Join(dir, logFileName)
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeTable(sb *strings.Builder, apps []appEntry) {
	headers := make([]string, appsPerRow)
	separators := make([]string, appsPerRow)
	for i := 0; i < appsPerRow; i++ {
		headers[i] = "App"
		separators[i] = "---"
	}
	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	sb.WriteString("| " + strings.Join(separators, " | ") + " |\n")

	for i := 0; i < len(apps); i += appsPerRow {
		row := make([]string, appsPerRow)
		for j := 0; j < appsPerRow; j++ {
			index := i + j
			if index >= len(apps) {
				row[j] = ""
				continue
			}
			row[j] = formatTableCell(apps[index])
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	if len(apps) == 0 {
		sb.WriteString("| " + strings.Repeat(" |", appsPerRow-1) + " |\n")
	}
}

func formatTableCell(app appEntry) string {
	name := escapeMarkdown(app.Name)
	icon := "(missing icon)"
	if app.HasIcon {
		icon = fmt.Sprintf("<img src=\"%s\" width=\"%d\" height=\"%d\" />", app.IconPath, saveIconSize, saveIconSize)
	}
	if name == "" {
		return icon
	}
	return icon + "<br>" + name
}

func escapeMarkdown(text string) string {
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", "\\|")
	return text
}

func splitAppErrors(err error) (int, error) {
	var errs []error
	missing := 0

	var collect func(error)
	collect = func(e error) {
		if e == nil {
			return
		}
		if errors.Is(e, deskact.ErrIconNotFound) {
			missing++
			return
		}
		type joiner interface {
			Unwrap() []error
		}
		if j, ok := e.(joiner); ok {
			for _, inner := range j.Unwrap() {
				collect(inner)
			}
			return
		}
		errs = append(errs, e)
	}

	collect(err)
	if len(errs) == 0 {
		return missing, nil
	}
	return missing, errors.Join(errs...)
}
