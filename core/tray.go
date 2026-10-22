package hfd

import "errors"

// TrayItem là một mục trong menu tray.
// Action là chuỗi tùy ý — khi người dùng bấm mục, app nhận event
// "tray" kèm payload {"action": Action} (lắng nghe bằng hfd.On).
type TrayItem struct {
	Label  string
	Action string
}

// TrayOptions mô tả icon + tooltip + menu của tray.
//
// IconPath: đường dẫn ảnh HOẶC tên icon trong theme hệ thống.
//  - rỗng           → icon mặc định "applications-system"
//  - "system-search" (không chứa '/', không đuôi file) → dùng làm IconName theme
//  - "/path/icon.png" → thư mục chứa được đưa vào IconThemePath,
//    tên file (bỏ đuôi) làm IconName — hỗ trợ fallback của nhiều host.
type TrayOptions struct {
	IconPath string
	Tooltip  string
	Items    []TrayItem
}

// trayBackend là interface tùy chọn — backend nào hỗ trợ tray thì implement.
type trayBackend interface {
	setTray(TrayOptions) error
}

// SetTray đăng ký tray icon (StatusNotifierItem trên Linux desktop).
//
// Linux: cần build với `-tags hfd_tray` (kích hoạt dep godbus).
// Click trái icon → event "tray" {action:"__activate"}; click mục menu →
// event "tray" {action:<Action của item>}. Host phải là shell hỗ trợ SNI
// (GNOME cần extension AppIndicator, KDE/LXQt/XFCE có sẵn).
func SetTray(opts TrayOptions) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetTray trước khi Run")
	}
	tb, ok := b.(trayBackend)
	if !ok {
		return errors.New("hfd: SetTray chưa hỗ trợ trên backend này (linux: build với -tags hfd_tray)")
	}
	return tb.setTray(opts)
}
