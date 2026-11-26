package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsVisible       = 0x10000000
	wsChild         = 0x40000000
	wsVScroll       = 0x00200000
	wsClipChildren  = 0x02000000
	wsTabStop       = 0x00010000
	bsOwnerDraw     = 0x0000000B
	bsAutoCheckBox  = 0x00000003
	ssCenterImage   = 0x00000200
	wmDestroy       = 0x0002
	wmSetFont       = 0x0030
	wmDrawItem      = 0x002B
	wmSetCursor     = 0x0020
	wmMouseMove     = 0x0200
	wmMouseLeave    = 0x02A3
	wmCommand       = 0x0111
	wmVScroll       = 0x0115
	wmMouseWheel    = 0x020A
	wmRebuild       = 0x8001
	swShow          = 5
	swpNoZOrder     = 0x0004
	swpFrameChanged = 0x0020
	bmSetCheck      = 0x00F1
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	comctl32 = windows.NewLazySystemDLL("comctl32.dll")

	procRegisterClassEx      = user32.NewProc("RegisterClassExW")
	procCreateWindowEx       = user32.NewProc("CreateWindowExW")
	procDefWindowProc        = user32.NewProc("DefWindowProcW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessage           = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessage      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procPostMessage          = user32.NewProc("PostMessageW")
	procSendMessage          = user32.NewProc("SendMessageW")
	procMessageBox           = user32.NewProc("MessageBoxW")
	procLoadCursor           = user32.NewProc("LoadCursorW")
	procLoadIcon             = user32.NewProc("LoadIconW")
	procLoadImage            = user32.NewProc("LoadImageW")
	procSetCursor            = user32.NewProc("SetCursor")
	procMoveWindow           = user32.NewProc("MoveWindow")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procSetWindowLongPtr     = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProc       = user32.NewProc("CallWindowProcW")
	procAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	procGetDpiForSystem      = user32.NewProc("GetDpiForSystem")
	procSetScrollInfo        = user32.NewProc("SetScrollInfo")
	procGetScrollInfo        = user32.NewProc("GetScrollInfo")
	procTrackMouseEvent      = user32.NewProc("TrackMouseEvent")
	procFillRect             = user32.NewProc("FillRect")
	procFrameRect            = user32.NewProc("FrameRect")
	procDrawText             = user32.NewProc("DrawTextW")
	procGetModuleHandle      = kernel32.NewProc("GetModuleHandleW")
	procCreateFont           = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush     = gdi32.NewProc("CreateSolidBrush")
	procSelectObject         = gdi32.NewProc("SelectObject")
	procSetBkMode            = gdi32.NewProc("SetBkMode")
	procSetTextColor         = gdi32.NewProc("SetTextColor")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
	procSetWindowTheme       = windows.NewLazySystemDLL("uxtheme.dll").NewProc("SetWindowTheme")
	procDwmSetAttribute      = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")

	mainProc       uintptr
	buttonProc     uintptr
	origButtonProc uintptr
	windowClass    *uint16
	ui             *windowState
)

type drawItem struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	_          uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     windows.Rect
	ItemData   uintptr
}

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	_       uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
	Private uint32
}

type trackMouse struct {
	Size  uint32
	Flags uint32
	Hwnd  uintptr
	Hover uint32
}

type scrollInfo struct {
	Size     uint32
	Mask     uint32
	Min      int32
	Max      int32
	Page     uint32
	Pos      int32
	TrackPos int32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type control struct {
	hwnd     uintptr
	id       int
	kind     string
	text     string
	value    string
	group    string
	x        int
	y        int
	w        int
	h        int
	selected bool
	hovered  bool
}

type windowState struct {
	hwnd         uintptr
	cfg          appConfig
	current      string
	envExists    bool
	settingsPath string
	hidden       map[string]bool
	dpi          int
	font         uintptr
	bold         uintptr
	background   uintptr
	textColor    uintptr
	brushes      [6]uintptr
	centered     bool
	controls     []control
	contentW     int
	contentH     int
	clientH      int
	scrollY      int
}

func showStartupError(title string, err error) {
	showError(0, title, err.Error())
}

func showError(owner uintptr, title, text string) {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	textPtr, _ := windows.UTF16PtrFromString(text)
	procMessageBox.Call(owner, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), 0x00000010|0x00010000)
}

func runWindow(cfg appConfig, current string, envExists bool, settingsPath string, hidden map[string]bool, settingsErr error) {
	if hidden == nil {
		hidden = map[string]bool{}
	}
	initWindowProcs()
	initCommonControls()

	state := &windowState{
		cfg:          cfg,
		current:      current,
		envExists:    envExists,
		settingsPath: settingsPath,
		hidden:       hidden,
		dpi:          systemDPI(),
	}
	state.font = createUIFont(state.dpi, 400)
	state.bold = createUIFont(state.dpi, 700)
	state.applyTheme()
	ui = state

	if err := registerWindowClass(state.background); err != nil {
		showStartupError(cfg.Title, err)
		return
	}
	if err := state.createMainWindow(); err != nil {
		showStartupError(cfg.Title, err)
		return
	}
	state.rebuild()
	procShowWindow.Call(state.hwnd, swShow)
	procUpdateWindow.Call(state.hwnd)
	if settingsErr != nil {
		showError(state.hwnd, cfg.Title, settingsErr.Error())
	}
	messageLoop()
}

func initWindowProcs() {
	if mainProc != 0 {
		return
	}
	mainProc = windows.NewCallback(mainWndProc)
	buttonProc = windows.NewCallback(buttonWndProc)
	windowClass, _ = windows.UTF16PtrFromString("SetEnvWindow")
}

func initCommonControls() {
	var icc struct {
		size uint32
		icc  uint32
	}
	icc.size = uint32(unsafe.Sizeof(icc))
	icc.icc = 0x00004000
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
}

func systemDPI() int {
	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi < 96 {
		return 96
	}
	return int(dpi)
}

func (s *windowState) px(n int) int {
	return n * s.dpi / 96
}

func createUIFont(dpi, weight int) uintptr {
	name, _ := windows.UTF16PtrFromString("Segoe UI")
	height := -dpi * 14 / 96
	font, _, _ := procCreateFont.Call(
		uintptr(height),
		0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(name)),
	)
	return font
}

func solidBrush(r, g, b byte) uintptr {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16)
	return brush
}

func systemMetric(index uintptr, fallback int) int {
	value, _, _ := procGetSystemMetrics.Call(index)
	if value == 0 {
		return fallback
	}
	return int(value)
}

func loadAppIcon(size int) uintptr {
	icon, _, _ := procLoadImage.Call(moduleHandle(), 1, 1, uintptr(size), uintptr(size), 0x8000)
	if icon == 0 {
		icon, _, _ = procLoadIcon.Call(0, 32512)
	}
	return icon
}

func moduleHandle() uintptr {
	handle, _, _ := procGetModuleHandle.Call(0)
	return handle
}

func registerWindowClass(background uintptr) error {
	cursor, _, _ := procLoadCursor.Call(0, 32512)
	class := wndClassEx{
		Style:      0x0003,
		WndProc:    mainProc,
		Instance:   moduleHandle(),
		Icon:       loadAppIcon(systemMetric(11, 32)),
		Cursor:     cursor,
		Background: background,
		ClassName:  windowClass,
		IconSm:     loadAppIcon(systemMetric(49, 16)),
	}
	class.Size = uint32(unsafe.Sizeof(class))
	atom, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		return fmt.Errorf("das Fenster konnte nicht registriert werden")
	}
	return nil
}

func (s *windowState) createMainWindow() error {
	title, _ := windows.UTF16PtrFromString(s.cfg.Title)
	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(windowClass)),
		uintptr(unsafe.Pointer(title)),
		wsCaption|wsSysMenu|wsMinimizeBox|wsClipChildren,
		100, 100, 400, 300,
		0, 0, moduleHandle(), 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("das Fenster konnte nicht geöffnet werden")
	}
	s.hwnd = hwnd
	if s.cfg.Dark {
		useDark := int32(1)
		procDwmSetAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&useDark)), 4)
		app, _ := windows.UTF16PtrFromString("DarkMode_Explorer")
		procSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(app)), 0)
	}
	return nil
}

func (s *windowState) applyTheme() {
	if s.cfg.Dark {
		s.textColor = colorRef(235, 235, 235)
		s.background = solidBrush(32, 32, 32)
		s.brushes = [6]uintptr{
			solidBrush(54, 54, 54),
			solidBrush(70, 70, 70),
			solidBrush(46, 130, 72),
			solidBrush(36, 112, 60),
			solidBrush(96, 96, 96),
			solidBrush(120, 190, 140),
		}
		return
	}
	s.textColor = colorRef(32, 32, 32)
	s.background = solidBrush(255, 255, 255)
	s.brushes = [6]uintptr{
		solidBrush(232, 232, 232),
		solidBrush(220, 220, 220),
		solidBrush(86, 186, 112),
		solidBrush(68, 164, 94),
		solidBrush(208, 208, 208),
		solidBrush(36, 112, 58),
	}
}

func colorRef(r, g, b byte) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}

func (s *windowState) rebuild() {
	for _, item := range s.controls {
		procDestroyWindow.Call(item.hwnd)
	}
	s.scrollY = 0
	s.controls = s.controlSpecs()
	s.placeMainWindow()
	s.createControls()
	if s.contentH > s.clientH {
		s.updateScroll()
	}
}

func (s *windowState) controlSpecs() []control {
	columns := normalizeColumns(s.cfg.Columns)
	bw, bh := s.px(136), s.px(68)
	gap, margin := s.px(8), s.px(8)
	s.contentW = margin*2 + columns*bw + (columns-1)*gap
	y := margin
	id := 100
	specs := make([]control, 0)

	clearH := s.px(32)
	specs = append(specs, control{
		id: id, kind: "clear", text: s.cfg.ClearLabel,
		x: margin, y: y, w: s.contentW - margin*2, h: clearH,
		selected: !s.envExists,
	})
	id++
	y += clearH + gap

	if s.cfg.hasNamedGroups() {
		width, height := s.px(150), s.px(30)
		specs = append(specs, control{
			id: id, kind: "showall", text: "Alle anzeigen",
			x: s.contentW - margin - width, y: y, w: width, h: height,
		})
		id++
		y += height + gap
	}

	visible := 0
	for _, group := range s.cfg.Groups {
		if group.Name != "" && s.hidden[group.Name] {
			continue
		}
		visible++
		if group.Name != "" {
			height := s.px(28)
			checkW := s.px(130)
			specs = append(specs, control{
				id: id, kind: "label", text: group.Name,
				x: margin, y: y, w: s.contentW - margin*2 - checkW - gap, h: height,
			})
			id++
			specs = append(specs, control{
				id: id, kind: "check", text: "ausblenden", group: group.Name,
				x: s.contentW - margin - checkW, y: y, w: checkW, h: height,
			})
			id++
			y += height + s.px(4)
		}
		for index, button := range group.Buttons {
			column := index % columns
			row := index / columns
			specs = append(specs, control{
				id: id, kind: "button", text: button.Anzeigename, value: button.Wert,
				x: margin + column*(bw+gap),
				y: y + row*(bh+gap),
				w: bw, h: bh,
				selected: s.envExists && button.Wert == s.current,
			})
			id++
		}
		rows := buttonRows(len(group.Buttons), columns)
		y += rows*bh + (rows-1)*gap + margin
	}
	if visible == 0 {
		specs = append(specs, control{
			id: id, kind: "note", text: "Alle Gruppen sind ausgeblendet.",
			x: margin, y: y, w: s.contentW - margin*2, h: s.px(28),
		})
		y += s.px(28) + margin
	}
	s.contentH = y
	return specs
}

func (s *windowState) maxClientHeight() int {
	maxH := s.px(80) + s.px(32) + s.px(8) + 4*(s.px(68)+s.px(8))
	area := workArea()
	available := int(area.Bottom-area.Top) - s.px(80)
	if available > s.px(120) && maxH > available {
		maxH = available
	}
	return maxH
}

func (s *windowState) placeMainWindow() {
	s.clientH = s.contentH
	maxH := s.maxClientHeight()
	if s.clientH > maxH {
		s.clientH = maxH
	}
	style := uintptr(wsCaption | wsSysMenu | wsMinimizeBox | wsClipChildren | wsVisible)
	if s.contentH > s.clientH {
		style |= wsVScroll
	}
	procSetWindowLongPtr.Call(s.hwnd, ^uintptr(15), style)
	width, height := outerSize(s.contentW, s.clientH, style)
	x, y := 0, 0
	if s.centered {
		var rect windows.Rect
		procGetWindowRect.Call(s.hwnd, uintptr(unsafe.Pointer(&rect)))
		x, y = int(rect.Left), int(rect.Top)
	} else {
		x, y = center(width, height)
		s.centered = true
	}
	procSetWindowPos.Call(s.hwnd, 0, uintptr(x), uintptr(y), uintptr(width), uintptr(height), swpNoZOrder|swpFrameChanged)
}

func outerSize(clientW, clientH int, style uintptr) (int, int) {
	rect := windows.Rect{Right: int32(clientW), Bottom: int32(clientH)}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&rect)), style, 0, 0)
	return int(rect.Right - rect.Left), int(rect.Bottom - rect.Top)
}

func workArea() windows.Rect {
	var rect windows.Rect
	procSystemParametersInfo.Call(0x0030, 0, uintptr(unsafe.Pointer(&rect)), 0)
	if rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		width, _, _ := procGetSystemMetrics.Call(0)
		height, _, _ := procGetSystemMetrics.Call(1)
		rect.Right = int32(width)
		rect.Bottom = int32(height)
	}
	return rect
}

func center(width, height int) (int, int) {
	area := workArea()
	x := int(area.Left) + (int(area.Right-area.Left)-width)/2
	y := int(area.Top) + (int(area.Bottom-area.Top)-height)/2
	if x < int(area.Left) {
		x = int(area.Left)
	}
	if y < int(area.Top) {
		y = int(area.Top)
	}
	return x, y
}

func (s *windowState) createControls() {
	instance := moduleHandle()
	for i := range s.controls {
		item := &s.controls[i]
		className := "Button"
		style := uintptr(wsChild | wsVisible | wsTabStop)
		switch item.kind {
		case "showall":
		case "check":
			style |= bsAutoCheckBox
		case "button", "clear":
			style |= bsOwnerDraw
		default:
			className = "Static"
			style = wsChild | wsVisible | ssCenterImage
		}
		classPtr, _ := windows.UTF16PtrFromString(className)
		textPtr, _ := windows.UTF16PtrFromString(item.text)
		item.hwnd, _, _ = procCreateWindowEx.Call(
			0,
			uintptr(unsafe.Pointer(classPtr)),
			uintptr(unsafe.Pointer(textPtr)),
			style,
			uintptr(item.x),
			uintptr(item.y),
			uintptr(item.w),
			uintptr(item.h),
			s.hwnd,
			uintptr(item.id),
			instance,
			0,
		)
		font := s.font
		if item.kind == "label" {
			font = s.bold
		}
		procSendMessage.Call(item.hwnd, wmSetFont, font, 1)
		if s.cfg.Dark && (item.kind == "showall" || item.kind == "check") {
			empty, _ := windows.UTF16PtrFromString("")
			procSetWindowTheme.Call(item.hwnd, uintptr(unsafe.Pointer(empty)), uintptr(unsafe.Pointer(empty)))
		}
		if item.kind == "button" || item.kind == "clear" {
			previous, _, _ := procSetWindowLongPtr.Call(item.hwnd, ^uintptr(3), buttonProc)
			if origButtonProc == 0 {
				origButtonProc = previous
			}
		}
		if item.kind == "showall" && !s.anyHidden() {
			procEnableWindow.Call(item.hwnd, 0)
		}
	}
}

func (s *windowState) anyHidden() bool {
	for _, group := range s.cfg.Groups {
		if group.Name != "" && s.hidden[group.Name] {
			return true
		}
	}
	return false
}

func (s *windowState) updateScroll() {
	info := scrollInfo{
		Mask: 7,
		Min:  0,
		Max:  int32(s.contentH - 1),
		Page: uint32(s.clientH),
		Pos:  int32(s.scrollY),
	}
	info.Size = uint32(unsafe.Sizeof(info))
	procSetScrollInfo.Call(s.hwnd, 1, uintptr(unsafe.Pointer(&info)), 1)
}

func (s *windowState) scrollTo(pos int) {
	maxPos := s.contentH - s.clientH
	if maxPos < 0 {
		maxPos = 0
	}
	if pos < 0 {
		pos = 0
	}
	if pos > maxPos {
		pos = maxPos
	}
	if pos == s.scrollY {
		return
	}
	s.scrollY = pos
	for _, item := range s.controls {
		procMoveWindow.Call(item.hwnd, uintptr(item.x), uintptr(item.y-s.scrollY), uintptr(item.w), uintptr(item.h), 1)
	}
	s.updateScroll()
}

func (s *windowState) onCommand(wParam uintptr) {
	if wParam>>16 != 0 {
		return
	}
	item := s.controlByID(int(wParam & 0xFFFF))
	if item == nil {
		return
	}
	switch item.kind {
	case "button":
		changed := !s.envExists || item.value != s.current
		if err := setUserEnv(s.cfg.Variable, item.value); err != nil {
			showError(s.hwnd, s.cfg.Title, err.Error())
			return
		}
		s.current = item.value
		s.envExists = true
		s.refreshSelection()
		if !changed {
			return
		}
		if err := applyProcessAction(s.cfg, item.value); err != nil {
			showError(s.hwnd, s.cfg.Title, "Der Arbeitsplatz wurde gesetzt.\n\n"+err.Error())
		}
	case "clear":
		changed := s.envExists
		if err := clearUserEnv(s.cfg.Variable); err != nil {
			showError(s.hwnd, s.cfg.Title, err.Error())
			return
		}
		s.current = ""
		s.envExists = false
		s.refreshSelection()
		if !changed {
			return
		}
		if err := applyProcessAction(s.cfg, ""); err != nil {
			showError(s.hwnd, s.cfg.Title, "Die Variable wurde gelöscht.\n\n"+err.Error())
		}
	case "check":
		s.hidden[item.group] = true
		if err := saveUserSettings(s.settingsPath, s.hidden); err != nil {
			delete(s.hidden, item.group)
			procSendMessage.Call(item.hwnd, bmSetCheck, 0, 0)
			showError(s.hwnd, s.cfg.Title, err.Error())
			return
		}
		procPostMessage.Call(s.hwnd, wmRebuild, 0, 0)
	case "showall":
		previous := s.hidden
		s.hidden = map[string]bool{}
		if err := saveUserSettings(s.settingsPath, s.hidden); err != nil {
			s.hidden = previous
			showError(s.hwnd, s.cfg.Title, err.Error())
			return
		}
		procPostMessage.Call(s.hwnd, wmRebuild, 0, 0)
	}
}

func (s *windowState) refreshSelection() {
	for i := range s.controls {
		switch s.controls[i].kind {
		case "button":
			s.controls[i].selected = s.envExists && s.controls[i].value == s.current
		case "clear":
			s.controls[i].selected = !s.envExists
		default:
			continue
		}
		procInvalidateRect.Call(s.controls[i].hwnd, 0, 1)
	}
}

func (s *windowState) controlByID(id int) *control {
	for i := range s.controls {
		if s.controls[i].id == id {
			return &s.controls[i]
		}
	}
	return nil
}

func (s *windowState) controlByHwnd(hwnd uintptr) *control {
	for i := range s.controls {
		if s.controls[i].hwnd == hwnd {
			return &s.controls[i]
		}
	}
	return nil
}

func (s *windowState) controlColor(hdc, brush uintptr) uintptr {
	procSetBkMode.Call(hdc, 1)
	procSetTextColor.Call(hdc, s.textColor)
	return brush
}

func (s *windowState) paintButton(lParam uintptr) {
	item := (*drawItem)(unsafe.Pointer(lParam))
	button := s.controlByHwnd(item.HwndItem)
	text := ""
	selected := false
	hovered := false
	if button != nil {
		text = button.text
		selected = button.selected
		hovered = button.hovered
	}
	pressed := item.ItemState&0x0001 != 0
	fill := s.brushes[0]
	border := s.brushes[4]
	if selected {
		fill = s.brushes[2]
		border = s.brushes[5]
		if hovered || pressed {
			fill = s.brushes[3]
		}
	} else if hovered || pressed {
		fill = s.brushes[1]
	}
	rect := item.RcItem
	procFillRect.Call(item.HDC, uintptr(unsafe.Pointer(&rect)), fill)
	procFrameRect.Call(item.HDC, uintptr(unsafe.Pointer(&rect)), border)
	procSetBkMode.Call(item.HDC, 1)
	procSetTextColor.Call(item.HDC, s.textColor)
	old, _, _ := procSelectObject.Call(item.HDC, s.font)
	rect.Left += 6
	rect.Right -= 6
	label, _ := windows.UTF16PtrFromString(text)
	procDrawText.Call(item.HDC, uintptr(unsafe.Pointer(label)), ^uintptr(0), uintptr(unsafe.Pointer(&rect)), 0x1|0x4|0x20|0x800|0x8000)
	procSelectObject.Call(item.HDC, old)
}

func mainWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if ui != nil && hwnd == ui.hwnd {
		switch msg {
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		case 0x0138:
			return ui.controlColor(wParam, ui.background)
		case 0x0135:
			if ui.cfg.Dark {
				brush := ui.background
				if item := ui.controlByHwnd(lParam); item != nil && item.kind == "showall" {
					brush = ui.brushes[0]
				}
				return ui.controlColor(wParam, brush)
			}
		case wmDrawItem:
			ui.paintButton(lParam)
			return 1
		case wmCommand:
			ui.onCommand(wParam)
			return 0
		case wmVScroll:
			ui.onScroll(wParam)
			return 0
		case wmMouseWheel:
			delta := int(int16(wParam >> 16))
			ui.scrollTo(ui.scrollY - delta/120*ui.px(68))
			return 0
		case wmRebuild:
			ui.rebuild()
			return 0
		}
	}
	result, _, _ := procDefWindowProc.Call(hwnd, msg, wParam, lParam)
	return result
}

func (s *windowState) onScroll(wParam uintptr) {
	code := wParam & 0xFFFF
	pos := s.scrollY
	switch code {
	case 0:
		pos -= s.px(40)
	case 1:
		pos += s.px(40)
	case 2:
		pos -= s.clientH
	case 3:
		pos += s.clientH
	case 4, 5:
		info := scrollInfo{Mask: 0x10}
		info.Size = uint32(unsafe.Sizeof(info))
		procGetScrollInfo.Call(s.hwnd, 1, uintptr(unsafe.Pointer(&info)))
		pos = int(info.TrackPos)
	default:
		return
	}
	s.scrollTo(pos)
}

func buttonWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if ui != nil {
		if button := ui.controlByHwnd(hwnd); button != nil {
			switch msg {
			case wmMouseMove:
				if !button.hovered {
					button.hovered = true
					track := trackMouse{Flags: 2, Hwnd: hwnd}
					track.Size = uint32(unsafe.Sizeof(track))
					procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track)))
					procInvalidateRect.Call(hwnd, 0, 1)
				}
			case wmMouseLeave:
				button.hovered = false
				procInvalidateRect.Call(hwnd, 0, 1)
			case wmSetCursor:
				cursor, _, _ := procLoadCursor.Call(0, 32649)
				procSetCursor.Call(cursor)
				return 1
			}
		}
	}
	if origButtonProc == 0 {
		return 0
	}
	result, _, _ := procCallWindowProc.Call(origButtonProc, hwnd, msg, wParam, lParam)
	return result
}

func messageLoop() {
	var msg winMsg
	for {
		result, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
