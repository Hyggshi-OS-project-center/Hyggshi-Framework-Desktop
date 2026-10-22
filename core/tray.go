package hfd

import "errors"

// TrayItem is an item in the tray menu.
// Action is an arbitrary string — when the user clicks the item, the app
// receives the "tray" event with the payload {"action": Action} (listen via hfd.On).
type TrayItem struct {
	Label  string
	Action string
}

// TrayOptions describes the tray icon + tooltip + menu.
//
// IconPath: an image path OR an icon name in the system theme.
//   - empty            → default icon "applications-system"
//   - "system-search" (no '/', no file extension) → used as the theme IconName
//   - "/path/icon.png" → the containing directory is added to IconThemePath,
//     the file name (without extension) as IconName — supports the fallback
//     of many hosts.
type TrayOptions struct {
	IconPath string
	Tooltip  string
	Items    []TrayItem
}

// trayBackend is an optional interface — whatever backend supports tray implements it.
type trayBackend interface {
	setTray(TrayOptions) error
}

// SetTray registers the tray icon (StatusNotifierItem on Linux desktops).
//
// Linux: needs to be built with `-tags hfd_tray` (enables the godbus dep).
// Left click on the icon → "tray" event {action:"__activate"}; click on a menu
// item → "tray" event {action:<Action of the item>}. The host must be an SNI-capable
// shell (GNOME needs the AppIndicator extension; KDE/LXQt/XFCE have it built in).
func SetTray(opts TrayOptions) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetTray before Run")
	}
	tb, ok := b.(trayBackend)
	if !ok {
		return errors.New("hfd: SetTray not supported on this backend (linux: build with -tags hfd_tray)")
	}
	return tb.setTray(opts)
}
