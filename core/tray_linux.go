//go:build linux && hfd_tray

package hfd

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

// Tray backend for Linux: StatusNotifierItem (SNI) + DBusMenu over DBus.
// Enabled with `-tags hfd_tray`; without the tag the core keeps 0 external deps.
//
// Signatures match the interfaces-xml of the AppIndicator extension:
//   GetLayout   (i,i,as) → (u, (ia{sv}av))
//   GetGroupProperties (ai, as) → (revision, a(ia{sv}))
//   Event       (i, s, v, u)

var (
	trayOnce sync.Once
	trayErr  error
	traySvc  *trayService
)

func (e *engine) setTray(opts TrayOptions) error {
	trayOnce.Do(func() {
		traySvc, trayErr = startTray(opts)
	})
	if trayErr != nil {
		return trayErr
	}
	// A later SetTray → dirty update of the current menu (bump revision + signal).
	traySvc.replaceItems(opts.Items)
	return nil
}

// trayAction pushes the "tray" event into JS.
func (e *engine) trayAction(action string) {
	_ = e.emit("tray", map[string]any{"action": action})
}

// ---- StatusNotifierItem -----------------------------------------------------

type sniTooltip struct {
	IconName string
	Pixmap   []sniIcon
	Title    string
	Body     string
}

type sniIcon struct {
	W, H int32
	Data []byte // ARGB32, byte order follows the system endianness
}

type trayService struct {
	mu    sync.Mutex
	conn  *dbus.Conn
	bus   string
	items []TrayItem
	byID  map[int32]string // menu item id → action
	title string
	icon  string
	theme string // IconThemePath (directory containing the icon file)
	rev   uint32
}

func startTray(opts TrayOptions) (*trayService, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("hfd: tray needs a DBus session: %w", err)
	}
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	bus := fmt.Sprintf("org.hfd.tray.p%d.h%x", os.Getpid(), buf)
	if _, err := conn.RequestName(bus, dbus.NameFlagDoNotQueue); err != nil {
		return nil, fmt.Errorf("hfd: request tray bus name: %w", err)
	}

	svc := &trayService{
		conn:  conn,
		bus:   bus,
		title: "HFD",
		icon:  "applications-system",
		byID:  map[int32]string{},
	}
	svc.mu.Lock()
	svc.setStaticLocked(opts)
	svc.mu.Unlock()

	// Hosts that register by bus name always probe the default object path
	// /StatusNotifierItem (the watcher's DEFAULT_ITEM_OBJECT_PATH).
	svcPath := dbus.ObjectPath("/StatusNotifierItem")
	menuPath := dbus.ObjectPath("/org/hfd/tray/menu")
	if err := conn.Export(svc, svcPath, "org.kde.StatusNotifierItem"); err != nil {
		return nil, fmt.Errorf("hfd: export SNI: %w", err)
	}
	if err := conn.Export(svc, svcPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, fmt.Errorf("hfd: export SNI props: %w", err)
	}
	menu := &trayMenu{svc: svc}
	if err := conn.Export(menu, menuPath, "com.canonical.dbusmenu"); err != nil {
		return nil, fmt.Errorf("hfd: export DBusMenu: %w", err)
	}
	if err := conn.Export(menu, menuPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, fmt.Errorf("hfd: export DBusMenu props: %w", err)
	}

	// Common watchers: KDE places the object at /org/kde/StatusNotifierWatcher,
	// gnome-shell (AppIndicator extension) at /StatusNotifierWatcher. Try both.
	watcherName := "org.kde.StatusNotifierWatcher"
	var regErr error
	for _, wpath := range []string{"/org/kde/StatusNotifierWatcher", "/StatusNotifierWatcher"} {
		watcher := conn.Object(watcherName, dbus.ObjectPath(wpath))
		regErr = watcher.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, bus).Err
		if regErr == nil {
			break
		}
	}
	if regErr != nil {
		return nil, fmt.Errorf("hfd: watcher rejected tray registration (desktop must support SNI): %w", regErr)
	}
	return svc, nil
}

// setStaticLocked computes icon/tooltip/item index (mu must be held).
func (s *trayService) setStaticLocked(opts TrayOptions) {
	if opts.Tooltip != "" {
		s.title = opts.Tooltip
	}
	p := opts.IconPath
	switch {
	case p == "":
		s.icon = "applications-system"
	case strings.Contains(p, "/"):
		s.theme = filepath.Dir(p)
		s.icon = strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	default:
		s.icon = strings.TrimSuffix(p, filepath.Ext(p)) // theme icon name
	}
	s.items = opts.Items
	s.buildIndexLocked()
}

func (s *trayService) buildIndexLocked() {
	s.byID = make(map[int32]string, len(s.items))
	for i, it := range s.items {
		s.byID[int32(i+1)] = it.Action
	}
}

func (s *trayService) replaceItems(items []TrayItem) {
	s.mu.Lock()
	s.items = items
	s.buildIndexLocked()
	s.rev++
	rev := s.rev
	s.mu.Unlock()

	menuPath := dbus.ObjectPath("/org/hfd/tray/menu")
	_ = s.conn.Emit(menuPath, "com.canonical.dbusmenu.LayoutUpdated", rev, int32(0))
}

// ---- org.freedesktop.DBus.Properties on /org/hfd/tray ----------------------

func (s *trayService) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	props, derr := s.GetAll(iface)
	if derr != nil {
		return dbus.Variant{}, derr
	}
	v, ok := props[prop]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{"hfd tray: no property " + prop})
	}
	return v, nil
}

func (s *trayService) Set(iface, prop string, val dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.Failed", []any{"hfd tray: Set not allowed for " + prop})
}

func (s *trayService) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	m := map[string]dbus.Variant{}
	switch iface {
	case "org.kde.StatusNotifierItem":
		s.mu.Lock()
		title, icon, theme := s.title, s.icon, s.theme
		s.mu.Unlock()
		m["Category"] = dbus.MakeVariant("ApplicationStatus")
		m["Id"] = dbus.MakeVariant("org.hfd")
		m["Title"] = dbus.MakeVariant(title)
		m["Status"] = dbus.MakeVariant("Active")
		m["IconName"] = dbus.MakeVariant(icon)
		if theme != "" {
			m["IconThemePath"] = dbus.MakeVariant([]string{theme})
		}
		m["ToolTip"] = dbus.MakeVariant(sniTooltip{Title: title})
		m["Menu"] = dbus.MakeVariant(dbus.ObjectPath("/org/hfd/tray/menu"))
		m["WindowId"] = dbus.MakeVariant(int32(0))
		return m, nil
	}
	return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{"hfd tray: unknown interface " + iface})
}

// ---- SNI method calls ---------------------------------------------------------

// Activate — the user left-clicked the icon.
func (s *trayService) Activate(x, y int32) *dbus.Error {
	if e, ok := curBackend.(*engine); ok {
		go e.trayAction("__activate")
	}
	return nil
}

func (s *trayService) SecondaryActivate(x, y int32) *dbus.Error           { return nil }
func (s *trayService) ContextMenu(x, y int32) *dbus.Error                 { return nil }
func (s *trayService) Scroll(delta int32, orientation string) *dbus.Error { return nil }

// ---- DBusMenu -----------------------------------------------------------------

func itemProps(label string) map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"label":   dbus.MakeVariant(label),
		"enabled": dbus.MakeVariant(true),
		"visible": dbus.MakeVariant(true),
	}
}

// menuChild matches the (ia{sv}av) signature of the layout node.
type menuChild struct {
	ID    int32
	Props map[string]dbus.Variant
	Kids  []dbus.Variant
}

// groupItem matches the (ia{sv}) of GetGroupProperties.
type groupItem struct {
	ID    int32
	Props map[string]dbus.Variant
}

type trayMenu struct {
	svc *trayService
}

func (t *trayMenu) propsAll() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"Version":       dbus.MakeVariant(uint32(3)),
		"TextDirection": dbus.MakeVariant("ltr"),
		"Status":        dbus.MakeVariant("normal"),
		"Id":            dbus.MakeVariant("/org/hfd/tray/menu"),
	}
}

func (t *trayMenu) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	props := t.propsAll()
	v, ok := props[prop]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{"hfd menu: no property " + prop})
	}
	return v, nil
}

func (t *trayMenu) Set(iface, prop string, val dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.Failed", []any{"hfd menu: Set not allowed for " + prop})
}

func (t *trayMenu) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	return t.propsAll(), nil
}

// AboutToShow(id) → false (the menu does not change any action).
func (t *trayMenu) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

// GetLayout(parent, depth, propNames) → (revision, layout (ia{sv}av)).
func (t *trayMenu) GetLayout(parent int32, depth int32, propNames []string) (uint32, dbus.Variant, *dbus.Error) {
	s := t.svc
	s.mu.Lock()
	rev := s.rev
	items := append([]TrayItem(nil), s.items...)
	s.mu.Unlock()

	kids := make([]dbus.Variant, 0, len(items))
	for i, it := range items {
		child := menuChild{ID: int32(i + 1), Props: itemProps(it.Label), Kids: []dbus.Variant{}}
		kids = append(kids, dbus.MakeVariant(child))
	}
	root := menuChild{
		ID:    0,
		Props: map[string]dbus.Variant{"children-display": dbus.MakeVariant("linear")},
		Kids:  kids,
	}
	return rev, dbus.MakeVariant(root), nil
}

// GetGroupProperties — libappindicator calls this after GetLayout.
func (t *trayMenu) GetGroupProperties(ids []int32, propNames []string) (uint32, []groupItem, *dbus.Error) {
	s := t.svc
	s.mu.Lock()
	rev := s.rev
	items := append([]TrayItem(nil), s.items...)
	s.mu.Unlock()

	out := []groupItem{}
	for _, id := range ids {
		if id == 0 {
			out = append(out, groupItem{0, map[string]dbus.Variant{
				"children-display": dbus.MakeVariant("linear"),
			}})
			continue
		}
		idx := int(id) - 1
		if idx < 0 || idx >= len(items) {
			continue
		}
		out = append(out, groupItem{int32(id), itemProps(items[idx].Label)})
	}
	return rev, out, nil
}

// Event(id, eventType, data, timestamp) — click on a menu item → emit event to JS.
func (t *trayMenu) Event(id int32, eventType string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventType != "clicked" {
		return nil
	}
	s := t.svc
	s.mu.Lock()
	action, ok := s.byID[id]
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if e, ok := curBackend.(*engine); ok {
		go e.trayAction(action)
	}
	return nil
}
