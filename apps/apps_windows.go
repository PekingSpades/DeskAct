//go:build windows

package apps

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
	xwindows "golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	slgpRawPath        = 0x0004
	stgmRead           = 0x00000000
	iconRequestSize    = 256
	rpcEChangedMode    = 0x80010106
	uninstallKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Uninstall`
	uninstallKeyPath32 = `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`
	shellAppsFolder    = "shell:AppsFolder\\"

	shcontfFolders    = 0x20
	shcontfNonFolders = 0x40

	sigdnNormalDisplay          = 0x00000000
	sigdnDesktopAbsoluteParsing = 0x80028000
)

var (
	clsidShellLink          = win.CLSID{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIShellLinkW          = win.IID{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIPersistFile         = win.IID{0x0000010B, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIShellFolder         = win.IID{0x000214E6, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	shell32DLL              = xwindows.NewLazySystemDLL("shell32.dll")
	procSHBindToObject      = shell32DLL.NewProc("SHBindToObject")
	procSHGetNameFromIDList = shell32DLL.NewProc("SHGetNameFromIDList")
	procILCombine           = shell32DLL.NewProc("ILCombine")
)

var (
	installerNameKeywords = []string{
		"setup",
		"installer",
		"install",
		"uninstall",
		"unins",
		"update",
		"updater",
		"patch",
		"hotfix",
		"upgrade",
		"repair",
		"driver",
		"drv",
		"bootstrap",
		"bootstrapper",
	}
	installerArgKeywords = []string{
		".msi",
		".msix",
		".appx",
		".appxbundle",
		" /i",
		" /x",
		"/uninstall",
		"/repair",
		"/update",
		" install",
		" uninstall",
		" setup",
	}
)

type IShellLinkW struct {
	LpVtbl *IShellLinkWVtbl
}

type IShellLinkWVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	GetPath             uintptr
	GetIDList           uintptr
	SetIDList           uintptr
	GetDescription      uintptr
	SetDescription      uintptr
	GetWorkingDirectory uintptr
	SetWorkingDirectory uintptr
	GetArguments        uintptr
	SetArguments        uintptr
	GetHotkey           uintptr
	SetHotkey           uintptr
	GetShowCmd          uintptr
	SetShowCmd          uintptr
	GetIconLocation     uintptr
	SetIconLocation     uintptr
	SetRelativePath     uintptr
	Resolve             uintptr
	SetPath             uintptr
}

type IPersistFile struct {
	LpVtbl *IPersistFileVtbl
}

type IPersistFileVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetClassID     uintptr
	IsDirty        uintptr
	Load           uintptr
	Save           uintptr
	SaveCompleted  uintptr
	GetCurFile     uintptr
}

type IShellFolder struct {
	LpVtbl *IShellFolderVtbl
}

type IShellFolderVtbl struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	ParseDisplayName uintptr
	EnumObjects      uintptr
	BindToObject     uintptr
	BindToStorage    uintptr
	CompareIDs       uintptr
	CreateViewObject uintptr
	GetAttributesOf  uintptr
	GetUIObjectOf    uintptr
	GetDisplayNameOf uintptr
	SetNameOf        uintptr
}

type IEnumIDList struct {
	LpVtbl *IEnumIDListVtbl
}

type IEnumIDListVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Next           uintptr
	Skip           uintptr
	Reset          uintptr
	Clone          uintptr
}

func (sl *IShellLinkW) QueryInterface(riid *win.IID, ppvObject *unsafe.Pointer) win.HRESULT {
	ret, _, _ := syscall.SyscallN(sl.LpVtbl.QueryInterface,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(ppvObject)),
	)
	return win.HRESULT(ret)
}

func (sl *IShellLinkW) Release() uint32 {
	ret, _, _ := syscall.SyscallN(sl.LpVtbl.Release,
		uintptr(unsafe.Pointer(sl)),
	)
	return uint32(ret)
}

func (sl *IShellLinkW) GetPath(pszFile *uint16, cchMaxPath int, pfd *xwindows.Win32finddata, fFlags uint32) win.HRESULT {
	ret, _, _ := syscall.SyscallN(sl.LpVtbl.GetPath,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(pszFile)),
		uintptr(cchMaxPath),
		uintptr(unsafe.Pointer(pfd)),
		uintptr(fFlags),
	)
	return win.HRESULT(ret)
}

func (sl *IShellLinkW) GetIconLocation(pszIconPath *uint16, cchIconPath int, piIcon *int32) win.HRESULT {
	ret, _, _ := syscall.SyscallN(sl.LpVtbl.GetIconLocation,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(pszIconPath)),
		uintptr(cchIconPath),
		uintptr(unsafe.Pointer(piIcon)),
	)
	return win.HRESULT(ret)
}

func (sl *IShellLinkW) GetArguments(pszArgs *uint16, cchMaxPath int) win.HRESULT {
	ret, _, _ := syscall.SyscallN(sl.LpVtbl.GetArguments,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(pszArgs)),
		uintptr(cchMaxPath),
	)
	return win.HRESULT(ret)
}

func (pf *IPersistFile) Release() uint32 {
	ret, _, _ := syscall.SyscallN(pf.LpVtbl.Release,
		uintptr(unsafe.Pointer(pf)),
	)
	return uint32(ret)
}

func (pf *IPersistFile) Load(pszFileName *uint16, mode uint32) win.HRESULT {
	ret, _, _ := syscall.SyscallN(pf.LpVtbl.Load,
		uintptr(unsafe.Pointer(pf)),
		uintptr(unsafe.Pointer(pszFileName)),
		uintptr(mode),
	)
	return win.HRESULT(ret)
}

func (sf *IShellFolder) Release() uint32 {
	ret, _, _ := syscall.SyscallN(sf.LpVtbl.Release,
		uintptr(unsafe.Pointer(sf)),
	)
	return uint32(ret)
}

func (sf *IShellFolder) EnumObjects(hwnd win.HWND, flags uint32, ppenum **IEnumIDList) win.HRESULT {
	ret, _, _ := syscall.SyscallN(sf.LpVtbl.EnumObjects,
		uintptr(unsafe.Pointer(sf)),
		uintptr(hwnd),
		uintptr(flags),
		uintptr(unsafe.Pointer(ppenum)),
	)
	return win.HRESULT(ret)
}

func (enum *IEnumIDList) Release() uint32 {
	ret, _, _ := syscall.SyscallN(enum.LpVtbl.Release,
		uintptr(unsafe.Pointer(enum)),
	)
	return uint32(ret)
}

func (enum *IEnumIDList) Next(celt uint32, rgelt *uintptr, fetched *uint32) win.HRESULT {
	ret, _, _ := syscall.SyscallN(enum.LpVtbl.Next,
		uintptr(unsafe.Pointer(enum)),
		uintptr(celt),
		uintptr(unsafe.Pointer(rgelt)),
		uintptr(unsafe.Pointer(fetched)),
	)
	return win.HRESULT(ret)
}

func DesktopApps() ([]AppInfo, error) {
	desktops, dirErr := desktopDirectories()
	if len(desktops) == 0 {
		return nil, dirErr
	}

	var apps []AppInfo
	var errs []error
	if dirErr != nil {
		errs = append(errs, dirErr)
	}

	seenNames := make(map[string]struct{})
	for _, desktop := range desktops {
		entries, err := os.ReadDir(desktop)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			lowerName := strings.ToLower(name)
			if _, exists := seenNames[lowerName]; exists {
				continue
			}

			ext := strings.ToLower(filepath.Ext(name))
			fullPath := filepath.Join(desktop, name)
			switch ext {
			case ".lnk":
				info, infoErr := appFromShortcut(fullPath)
				if errors.Is(infoErr, errSkipEntry) {
					continue
				}
				if infoErr != nil {
					errs = append(errs, infoErr)
				}
				if info.Name != "" {
					apps = append(apps, info)
					seenNames[lowerName] = struct{}{}
				}
			case ".exe":
				if shouldSkipDesktopExe(fullPath) {
					continue
				}
				info, infoErr := appFromExe(fullPath)
				if infoErr != nil {
					errs = append(errs, infoErr)
				}
				if info.Name != "" {
					apps = append(apps, info)
					seenNames[lowerName] = struct{}{}
				}
			}
		}
	}

	sortApps(apps)
	return apps, joinErrors(errs)
}

func InstalledApps() ([]AppInfo, error) {
	roots := []registry.Key{
		registry.CURRENT_USER,
		registry.LOCAL_MACHINE,
	}
	subPaths := []string{
		uninstallKeyPath,
		uninstallKeyPath32,
	}

	seen := make(map[string]struct{})
	var apps []AppInfo
	var errs []error
	for _, root := range roots {
		for _, subPath := range subPaths {
			items, err := appsFromRegistry(root, subPath)
			if err != nil {
				errs = append(errs, err)
			}
			for _, item := range items {
				key := appKey(item)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				apps = append(apps, item)
			}
		}
	}

	appsFolderItems, folderErr := appsFromAppsFolder(true)
	if folderErr != nil {
		errs = append(errs, folderErr)
	}
	for _, item := range appsFolderItems {
		key := appKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		apps = append(apps, item)
	}

	sortApps(apps)
	return apps, joinErrors(errs)
}

func appFromExe(path string) (AppInfo, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	icon, err := iconFromFile(path, 0, false)
	return AppInfo{
		Name: name,
		Path: path,
		Icon: icon,
	}, err
}

func appFromShortcut(path string) (AppInfo, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	target, iconPath, iconIndex, args, resolveErr := resolveShortcut(path)
	if target == "" {
		target = path
	}

	if appsFolderPath := extractAppsFolderPath(target, args); appsFolderPath != "" {
		info, infoErr := appInfoFromShellItemPath(appsFolderPath)
		if info.Name == "" {
			info.Name = name
		}
		if info.Path == "" {
			info.Path = appsFolderPath
		}
		if info.Icon == nil {
			source := iconPath
			if source == "" {
				source = target
			}
			source = resolveRelativePath(normalizePath(source), path)
			if source != "" {
				fallback, fallbackErr := iconFromFile(source, iconIndex, iconPath != "")
				if fallbackErr == nil {
					info.Icon = fallback
					infoErr = nil
				} else if infoErr != nil {
					infoErr = joinErrors([]error{infoErr, fallbackErr})
				} else {
					infoErr = fallbackErr
				}
			}
		}

		if resolveErr != nil && infoErr != nil {
			return info, joinErrors([]error{resolveErr, infoErr})
		}
		if resolveErr != nil {
			return info, resolveErr
		}
		return info, infoErr
	}

	if isInstallerTarget(target, args) {
		return AppInfo{}, errSkipEntry
	}

	source := iconPath
	if source == "" {
		source = target
	}
	source = resolveRelativePath(normalizePath(source), path)

	icon, iconErr := iconFromFile(source, iconIndex, iconPath != "")
	if iconErr != nil && source != path {
		fallback, fallbackErr := iconFromFile(path, 0, false)
		if fallbackErr == nil {
			icon = fallback
			iconErr = nil
		} else {
			iconErr = joinErrors([]error{iconErr, fallbackErr})
		}
	}

	info := AppInfo{
		Name: name,
		Path: target,
		Icon: icon,
	}

	if resolveErr != nil && iconErr != nil {
		return info, joinErrors([]error{resolveErr, iconErr})
	}
	if resolveErr != nil {
		return info, resolveErr
	}
	return info, iconErr
}

func appsFromRegistry(root registry.Key, path string) ([]AppInfo, error) {
	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer key.Close()

	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	var apps []AppInfo
	var errs []error
	for _, name := range names {
		sub, err := registry.OpenKey(key, name, registry.READ)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		info, infoErr := appFromUninstallKey(sub)
		sub.Close()
		if errors.Is(infoErr, errSkipEntry) {
			continue
		}
		if infoErr != nil {
			errs = append(errs, infoErr)
		}
		if info.Name == "" {
			continue
		}
		apps = append(apps, info)
	}

	return apps, joinErrors(errs)
}

var errSkipEntry = errors.New("skip entry")

func appFromUninstallKey(key registry.Key) (AppInfo, error) {
	displayName, _, err := key.GetStringValue("DisplayName")
	if err != nil || strings.TrimSpace(displayName) == "" {
		return AppInfo{}, errSkipEntry
	}
	if isSystemComponent(key) {
		return AppInfo{}, errSkipEntry
	}

	displayName = strings.TrimSpace(displayName)
	displayIcon, _, _ := key.GetStringValue("DisplayIcon")
	installLocation, _, _ := key.GetStringValue("InstallLocation")
	uninstallString, _, _ := key.GetStringValue("UninstallString")
	quietUninstallString, _, _ := key.GetStringValue("QuietUninstallString")
	modifyPath, _, _ := key.GetStringValue("ModifyPath")

	iconPath, iconIndex, hasIndex := parseIconLocation(displayIcon)
	iconPath = normalizePath(iconPath)
	installLocation = normalizePath(installLocation)

	appPath := iconPath
	if appPath == "" {
		appPath = normalizePath(extractExeFromCommandLine(uninstallString))
		if appPath == "" {
			appPath = normalizePath(extractExeFromCommandLine(quietUninstallString))
		}
		if appPath == "" {
			appPath = normalizePath(extractExeFromCommandLine(modifyPath))
		}
	}
	if appPath == "" {
		appPath = pickExeFromInstallLocation(installLocation, displayName)
		if appPath == "" {
			appPath = installLocation
		}
	}

	var icon *image.RGBA
	var iconErr error
	if iconPath != "" {
		icon, iconErr = iconFromFile(iconPath, iconIndex, hasIndex)
	} else if appPath != "" {
		icon, iconErr = iconFromFile(appPath, 0, false)
	} else {
		iconErr = ErrIconNotFound
	}

	return AppInfo{
		Name: displayName,
		Path: appPath,
		Icon: icon,
	}, iconErr
}

func appsFromAppsFolder(onlyPackaged bool) ([]AppInfo, error) {
	hr := win.CoInitializeEx(nil, win.COINIT_APARTMENTTHREADED)
	uninit := hr == win.S_OK || hr == win.S_FALSE
	if win.FAILED(hr) && uint32(hr) != rpcEChangedMode {
		return nil, errors.New("CoInitializeEx failed")
	}
	if uninit {
		defer win.CoUninitialize()
	}

	pidlApps, err := parseShellItemPIDL(shellAppsFolder)
	if err != nil {
		return nil, err
	}
	defer win.CoTaskMemFree(pidlApps)

	var folder *IShellFolder
	hr = shBindToObject(pidlApps, &iidIShellFolder, unsafe.Pointer(&folder))
	if win.FAILED(hr) || folder == nil {
		return nil, errors.New("SHBindToObject failed")
	}
	defer folder.Release()

	var enum *IEnumIDList
	hr = folder.EnumObjects(0, shcontfFolders|shcontfNonFolders, &enum)
	if win.FAILED(hr) || enum == nil {
		return nil, errors.New("EnumObjects failed")
	}
	defer enum.Release()

	seen := make(map[string]struct{})
	var apps []AppInfo
	var errs []error
	for {
		var itemPIDL uintptr
		var fetched uint32
		hr = enum.Next(1, &itemPIDL, &fetched)
		if hr == win.S_FALSE || fetched == 0 {
			break
		}
		if win.FAILED(hr) {
			errs = append(errs, errors.New("EnumObjects next failed"))
			break
		}
		if itemPIDL == 0 {
			continue
		}

		fullPIDL := ilCombine(pidlApps, itemPIDL)
		win.CoTaskMemFree(itemPIDL)
		if fullPIDL == 0 {
			continue
		}

		info, infoErr := appInfoFromPIDL(fullPIDL)
		win.CoTaskMemFree(fullPIDL)
		if infoErr != nil {
			errs = append(errs, infoErr)
		}
		if info.Name == "" {
			continue
		}
		if onlyPackaged && !isPackagedAppPath(info.Path) {
			continue
		}
		key := appKey(info)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		apps = append(apps, info)
	}

	return apps, joinErrors(errs)
}

func appInfoFromShellItemPath(path string) (AppInfo, error) {
	pidl, err := parseShellItemPIDL(path)
	if err != nil {
		return AppInfo{}, err
	}
	defer win.CoTaskMemFree(pidl)
	return appInfoFromPIDL(pidl)
}

func appInfoFromPIDL(pidl uintptr) (AppInfo, error) {
	displayName, nameErr := nameFromPIDL(pidl, sigdnNormalDisplay)
	parsingName, parseErr := nameFromPIDL(pidl, sigdnDesktopAbsoluteParsing)
	icon, iconErr := iconFromPIDL(pidl)

	if displayName == "" {
		displayName = parsingName
	}
	if parsingName == "" {
		parsingName = displayName
	}

	info := AppInfo{
		Name: displayName,
		Path: parsingName,
		Icon: icon,
	}

	var errs []error
	if nameErr != nil {
		errs = append(errs, nameErr)
	}
	if parseErr != nil {
		errs = append(errs, parseErr)
	}
	if iconErr != nil {
		errs = append(errs, iconErr)
	}
	return info, joinErrors(errs)
}

func iconFromPIDL(pidl uintptr) (*image.RGBA, error) {
	var sfi win.SHFILEINFO
	flags := uint32(win.SHGFI_PIDL | win.SHGFI_ICON | win.SHGFI_LARGEICON)
	if win.SHGetFileInfo((*uint16)(unsafe.Pointer(pidl)), 0, &sfi, uint32(unsafe.Sizeof(sfi)), flags) == 0 || sfi.HIcon == 0 {
		return nil, ErrIconNotFound
	}
	defer win.DestroyIcon(sfi.HIcon)
	return hiconToRGBA(sfi.HIcon)
}

func nameFromPIDL(pidl uintptr, sigdn uint32) (string, error) {
	var psz *uint16
	ret, _, _ := procSHGetNameFromIDList.Call(pidl, uintptr(sigdn), uintptr(unsafe.Pointer(&psz)))
	hr := win.HRESULT(ret)
	if win.FAILED(hr) || psz == nil {
		return "", errors.New("SHGetNameFromIDList failed")
	}
	defer win.CoTaskMemFree(uintptr(unsafe.Pointer(psz)))
	return xwindows.UTF16PtrToString(psz), nil
}

func parseShellItemPIDL(path string) (uintptr, error) {
	if strings.TrimSpace(path) == "" {
		return 0, errors.New("shell item path empty")
	}
	ptr, err := xwindows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var pidl uintptr
	hr := win.SHParseDisplayName(ptr, 0, &pidl, 0, nil)
	if win.FAILED(hr) || pidl == 0 {
		return 0, errors.New("SHParseDisplayName failed")
	}
	return pidl, nil
}

func isPackagedAppPath(path string) bool {
	return hasAppsFolderItem(path) && strings.Contains(path, "!")
}

func extractAppsFolderPath(target string, args string) string {
	if path := findAppsFolderPath(target); path != "" {
		if hasAppsFolderItem(path) {
			return path
		}
	}
	if path := findAppsFolderPath(args); path != "" {
		if hasAppsFolderItem(path) {
			return path
		}
	}
	return ""
}

func hasAppsFolderItem(path string) bool {
	lower := strings.ToLower(path)
	prefixLower := strings.ToLower(shellAppsFolder)
	return strings.HasPrefix(lower, prefixLower) && len(path) > len(shellAppsFolder)
}

func findAppsFolderPath(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	lower := strings.ToLower(source)
	idx := strings.Index(lower, "shell:appsfolder")
	if idx == -1 {
		return ""
	}
	tail := strings.TrimLeft(source[idx:], " \"'")
	if tail == "" {
		return ""
	}
	end := len(tail)
	for i, r := range tail {
		if r == '"' || r == '\'' || r == ' ' || r == '\t' {
			end = i
			break
		}
	}
	tail = tail[:end]
	return normalizeAppsFolderPath(tail)
}

func normalizeAppsFolderPath(path string) string {
	path = strings.TrimSpace(strings.Trim(path, "\""))
	lower := strings.ToLower(path)
	prefixLower := strings.ToLower(shellAppsFolder)
	if strings.HasPrefix(lower, prefixLower) {
		return shellAppsFolder + path[len(prefixLower):]
	}
	if strings.EqualFold(path, strings.TrimSuffix(shellAppsFolder, "\\")) {
		return shellAppsFolder
	}
	return path
}

func shouldSkipDesktopExe(path string) bool {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	return isInstallerName(base)
}

func isInstallerTarget(target string, args string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if hasAppsFolderItem(target) || hasAppsFolderItem(args) {
		return false
	}
	base := strings.ToLower(filepath.Base(target))
	if base == "msiexec.exe" || base == "rundll32.exe" {
		return true
	}
	baseName := strings.TrimSuffix(base, filepath.Ext(base))
	if isInstallerName(baseName) {
		return true
	}
	return hasInstallerArgs(args)
}

func isInstallerName(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	for _, keyword := range installerNameKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func hasInstallerArgs(args string) bool {
	if args == "" {
		return false
	}
	lower := strings.ToLower(args)
	for _, keyword := range installerArgKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func shBindToObject(pidl uintptr, riid *win.IID, ppv unsafe.Pointer) win.HRESULT {
	ret, _, _ := procSHBindToObject.Call(
		0,
		pidl,
		0,
		uintptr(unsafe.Pointer(riid)),
		uintptr(ppv),
	)
	return win.HRESULT(ret)
}

func ilCombine(pidl1 uintptr, pidl2 uintptr) uintptr {
	ret, _, _ := procILCombine.Call(pidl1, pidl2)
	return ret
}

func isSystemComponent(key registry.Key) bool {
	value, _, err := key.GetIntegerValue("SystemComponent")
	return err == nil && value == 1
}

func pickExeFromInstallLocation(dir string, displayName string) string {
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	nameLower := strings.ToLower(displayName)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".exe" {
			continue
		}
		base := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if base == nameLower {
			return filepath.Join(dir, entry.Name())
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == ".exe" {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

func parseIconLocation(raw string) (string, int, bool) {
	path := cleanPath(raw)
	if path == "" {
		return "", 0, false
	}
	if idx := strings.LastIndex(path, ","); idx != -1 {
		tail := strings.TrimSpace(path[idx+1:])
		if tail != "" {
			if val, err := strconv.Atoi(tail); err == nil {
				return strings.TrimSpace(path[:idx]), val, true
			}
		}
	}
	return path, 0, false
}

func cleanPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	return s
}

func normalizePath(raw string) string {
	s := cleanPath(raw)
	if s == "" {
		return ""
	}
	expanded, err := expandEnv(s)
	if err == nil {
		s = expanded
	}
	s = strings.Trim(s, "\"")
	return s
}

func extractExeFromCommandLine(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	cmd = strings.TrimSpace(strings.TrimPrefix(cmd, "@"))
	var path string
	if strings.HasPrefix(cmd, "\"") {
		end := strings.Index(cmd[1:], "\"")
		if end == -1 {
			return ""
		}
		path = cmd[1 : 1+end]
	} else {
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			return ""
		}
		path = fields[0]
	}
	path = normalizePath(path)
	if path == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(path))
	if base == "msiexec.exe" || base == "rundll32.exe" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(path), ".exe") {
		return ""
	}
	return path
}

func resolveRelativePath(path string, base string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(base), path)
}

func appKey(app AppInfo) string {
	if app.Path != "" {
		return strings.ToLower(app.Path)
	}
	return strings.ToLower(app.Name)
}

func desktopDirectory() (string, error) {
	path, err := desktopDirectoryByCSIDL(win.CSIDL_DESKTOPDIRECTORY)
	if err != nil {
		return "", err
	}
	return path, nil
}

func desktopDirectories() ([]string, error) {
	var dirs []string
	var errs []error

	userDir, err := desktopDirectoryByCSIDL(win.CSIDL_DESKTOPDIRECTORY)
	if err == nil && userDir != "" {
		dirs = append(dirs, userDir)
	} else if err != nil {
		errs = append(errs, err)
	}

	publicDir, err := desktopDirectoryByCSIDL(win.CSIDL_COMMON_DESKTOPDIRECTORY)
	if err == nil && publicDir != "" {
		dirs = append(dirs, publicDir)
	} else if err != nil {
		errs = append(errs, err)
	}

	if len(dirs) == 0 {
		return nil, joinErrors(errs)
	}
	return dirs, joinErrors(errs)
}

func desktopDirectoryByCSIDL(csidl win.CSIDL) (string, error) {
	var buf [win.MAX_PATH]uint16
	if !win.SHGetSpecialFolderPath(0, &buf[0], csidl, false) {
		return "", errors.New("desktop directory not found")
	}
	return xwindows.UTF16ToString(buf[:]), nil
}

func resolveShortcut(path string) (string, string, int, string, error) {
	hr := win.CoInitializeEx(nil, win.COINIT_APARTMENTTHREADED)
	uninit := hr == win.S_OK || hr == win.S_FALSE
	if win.FAILED(hr) && uint32(hr) != rpcEChangedMode {
		return "", "", 0, "", errors.New("CoInitializeEx failed")
	}
	if uninit {
		defer win.CoUninitialize()
	}

	var sl *IShellLinkW
	var unk unsafe.Pointer
	hr = win.CoCreateInstance(&clsidShellLink, nil, win.CLSCTX_INPROC_SERVER, &iidIShellLinkW, &unk)
	if win.FAILED(hr) || unk == nil {
		return "", "", 0, "", errors.New("CoCreateInstance failed")
	}
	sl = (*IShellLinkW)(unk)
	defer sl.Release()

	var pf *IPersistFile
	hr = sl.QueryInterface(&iidIPersistFile, &unk)
	if win.FAILED(hr) || unk == nil {
		return "", "", 0, "", errors.New("QueryInterface failed")
	}
	pf = (*IPersistFile)(unk)
	defer pf.Release()

	lnkPath, err := xwindows.UTF16PtrFromString(path)
	if err != nil {
		return "", "", 0, "", err
	}
	hr = pf.Load(lnkPath, stgmRead)
	if win.FAILED(hr) {
		return "", "", 0, "", errors.New("shortcut load failed")
	}

	targetBuf := make([]uint16, win.MAX_PATH)
	hr = sl.GetPath(&targetBuf[0], win.MAX_PATH, nil, slgpRawPath)
	if win.FAILED(hr) {
		return "", "", 0, "", errors.New("shortcut target read failed")
	}
	target := xwindows.UTF16ToString(targetBuf)

	iconBuf := make([]uint16, win.MAX_PATH)
	var iconIndex int32
	hr = sl.GetIconLocation(&iconBuf[0], win.MAX_PATH, &iconIndex)
	if win.FAILED(hr) {
		return target, "", 0, "", errors.New("shortcut icon read failed")
	}
	iconPath := xwindows.UTF16ToString(iconBuf)
	iconPath = resolveRelativePath(iconPath, path)

	argsBuf := make([]uint16, win.MAX_PATH)
	args := ""
	if hr = sl.GetArguments(&argsBuf[0], win.MAX_PATH); !win.FAILED(hr) {
		args = xwindows.UTF16ToString(argsBuf)
	}

	return target, iconPath, int(iconIndex), args, nil
}

func iconFromFile(path string, index int, hasIndex bool) (*image.RGBA, error) {
	path = normalizePath(path)
	if path == "" {
		return nil, ErrIconNotFound
	}

	var hIcon win.HICON
	var err error
	if hasIndex {
		hIcon, err = extractIconByIndex(path, index)
	}
	if hIcon == 0 {
		hIcon, err = extractIconByFile(path)
	}
	if hIcon == 0 {
		if err == nil {
			err = ErrIconNotFound
		}
		return nil, err
	}
	defer win.DestroyIcon(hIcon)

	img, err := hiconToRGBA(hIcon)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func extractIconByIndex(path string, index int) (win.HICON, error) {
	ptr, err := xwindows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var hIcon win.HICON
	hr := win.SHDefExtractIcon(ptr, int32(index), 0, &hIcon, nil, win.MAKELONG(uint16(iconRequestSize), uint16(iconRequestSize)))
	if win.FAILED(hr) || hIcon == 0 {
		return 0, errors.New("SHDefExtractIcon failed")
	}
	return hIcon, nil
}

func extractIconByFile(path string) (win.HICON, error) {
	ptr, err := xwindows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var sfi win.SHFILEINFO
	flags := uint32(win.SHGFI_ICON | win.SHGFI_LARGEICON)
	if win.SHGetFileInfo(ptr, 0, &sfi, uint32(unsafe.Sizeof(sfi)), flags) == 0 || sfi.HIcon == 0 {
		return 0, errors.New("SHGetFileInfo failed")
	}
	return sfi.HIcon, nil
}

func hiconToRGBA(hIcon win.HICON) (*image.RGBA, error) {
	var info win.ICONINFO
	hasInfo := win.GetIconInfo(hIcon, &info)
	if hasInfo {
		defer win.DeleteObject(win.HGDIOBJ(info.HbmMask))
		if info.HbmColor != 0 {
			defer win.DeleteObject(win.HGDIOBJ(info.HbmColor))
		}
	}

	width := 0
	height := 0
	if hasInfo {
		hBmp := info.HbmColor
		if hBmp == 0 {
			hBmp = info.HbmMask
		}
		var bmp win.BITMAP
		if win.GetObject(win.HGDIOBJ(hBmp), unsafe.Sizeof(bmp), unsafe.Pointer(&bmp)) != 0 {
			width = int(bmp.BmWidth)
			height = int(bmp.BmHeight)
			if info.HbmColor == 0 && height > 1 {
				height = height / 2
			}
		}
	}
	if width <= 0 || height <= 0 {
		width = int(win.GetSystemMetrics(win.SM_CXICON))
		height = int(win.GetSystemMetrics(win.SM_CYICON))
		if width <= 0 || height <= 0 {
			return nil, errors.New("icon size invalid")
		}
	}

	hdc := win.CreateCompatibleDC(0)
	if hdc == 0 {
		return nil, errors.New("CreateCompatibleDC failed")
	}
	defer win.DeleteDC(hdc)

	var header win.BITMAPINFOHEADER
	header.BiSize = uint32(unsafe.Sizeof(header))
	header.BiPlanes = 1
	header.BiBitCount = 32
	header.BiWidth = int32(width)
	header.BiHeight = int32(-height)
	header.BiCompression = win.BI_RGB

	var bits unsafe.Pointer
	hBmp := win.CreateDIBSection(hdc, &header, win.DIB_RGB_COLORS, &bits, 0, 0)
	if hBmp == 0 {
		return nil, errors.New("CreateDIBSection failed")
	}
	defer win.DeleteObject(win.HGDIOBJ(hBmp))

	old := win.SelectObject(hdc, win.HGDIOBJ(hBmp))
	if old == 0 {
		return nil, errors.New("SelectObject failed")
	}
	defer win.SelectObject(hdc, old)

	if !win.DrawIconEx(hdc, 0, 0, hIcon, int32(width), int32(height), 0, 0, win.DI_NORMAL) {
		return nil, errors.New("DrawIconEx failed")
	}

	pixelCount := width * height * 4
	if pixelCount <= 0 {
		return nil, errors.New("icon buffer invalid")
	}

	src := unsafe.Slice((*byte)(bits), pixelCount)
	pix := make([]byte, pixelCount)
	copy(pix, src)

	hasAlpha := false
	for i := 0; i < len(pix); i += 4 {
		b, g, r, a := pix[i], pix[i+1], pix[i+2], pix[i+3]
		pix[i], pix[i+1], pix[i+2], pix[i+3] = r, g, b, a
		if a != 0 {
			hasAlpha = true
		}
	}

	if !hasAlpha && hasInfo && info.HbmMask != 0 {
		if maskErr := applyMaskAlpha(pix, width, height, info.HbmMask); maskErr == nil {
			hasAlpha = true
		}
	}
	if !hasAlpha {
		for i := 3; i < len(pix); i += 4 {
			pix[i] = 0xFF
		}
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}, nil
}

func applyMaskAlpha(pix []byte, width, height int, mask win.HBITMAP) error {
	if width <= 0 || height <= 0 {
		return errors.New("mask size invalid")
	}
	rowSize := ((width + 31) / 32) * 4
	if rowSize <= 0 {
		return errors.New("mask row size invalid")
	}

	hdc := win.CreateCompatibleDC(0)
	if hdc == 0 {
		return errors.New("CreateCompatibleDC failed")
	}
	defer win.DeleteDC(hdc)

	var bmi win.BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = int32(width)
	bmi.BmiHeader.BiHeight = int32(-height)
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 1
	bmi.BmiHeader.BiCompression = win.BI_RGB

	maskBits := make([]byte, rowSize*height)
	if win.GetDIBits(hdc, mask, 0, uint32(height), &maskBits[0], &bmi, win.DIB_RGB_COLORS) == 0 {
		return errors.New("GetDIBits failed")
	}

	for y := 0; y < height; y++ {
		row := maskBits[y*rowSize:]
		for x := 0; x < width; x++ {
			byteIndex := x / 8
			bit := 7 - (x % 8)
			if ((row[byteIndex] >> bit) & 1) == 1 {
				pix[(y*width+x)*4+3] = 0
			} else {
				pix[(y*width+x)*4+3] = 0xFF
			}
		}
	}

	return nil
}

func expandEnv(path string) (string, error) {
	if !strings.Contains(path, "%") {
		return path, nil
	}
	src, err := xwindows.UTF16PtrFromString(path)
	if err != nil {
		return path, err
	}
	buf := make([]uint16, 256)
	for {
		n, err := xwindows.ExpandEnvironmentStrings(src, &buf[0], uint32(len(buf)))
		if err == nil {
			return xwindows.UTF16ToString(buf[:n]), nil
		}
		if err != xwindows.ERROR_INSUFFICIENT_BUFFER {
			return path, err
		}
		buf = make([]uint16, n)
	}
}
