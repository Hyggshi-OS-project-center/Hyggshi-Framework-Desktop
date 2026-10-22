// Package hfd — Go runtime của Hyggshi Framework Desktop (HFD).
//
// Kiến trúc theo spec của Hyggshi:
//   - Go:              runtime + native API (gói này).
//   - TypeScript:      CLI/devtools (phase 2).
//   - React và Vue:    frontend tùy chọn.
//   - GTK + WebKitGTK: cửa sổ/webview trên Linux.
//   - IPC/bindings:    script message handler với native reply (JS→Go)
//                      + evaluate_javascript (Go→JS).
package hfd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"sync"
)

// Handler chạy khi JS gọi hfd.call(name, args).
// `args` là JSON thô của payload (string "null" nếu JS không truyền gì).
// Giá trị trả về sẽ được chạy qua json.Marshal và nộp ngược cho JS; error
// khiến promise trên JS side reject với message error đó.
type Handler func(args string) (any, error)

// Options cấu hình app.
type Options struct {
	// Tên cửa sổ. Mặc định "HFD app".
	Title string
	// Kích cửa sổ. Mặc định 1024×768.
	Width  int
	Height int
	// Assets chứa frontend build (root phải chứa index.html).
	// Dùng fs.Sub(embedFS, "frontend/dist") nếu embed có prefix.
	// Nil → chỉ phục vụ runtime JS + trang index mặc định.
	Assets fs.FS
	// Debug bật WebKit Inspector (chuột phải → Inspect Element).
	Debug bool
	// DevURL cho dev mode (vd "http://localhost:5173" — vite dev server).
	// Nếu rỗng, CLI có thể đặt qua env HFD_DEV_URL.
	DevURL string
	// Menu bar GTK (linux) — item bấm → JS nhận event "menu" {action}.
	MenuLabel string // mặc định "Menu"
	Menu      []MenuItem
	// AllowOrigins — pattern URI được phép chạy bridge IPC `window.hfd`.
	// pattern glob với '*' bất kỳ chuỗi: "hfd://app/*", "http://localhost:5173/*".
	// Nil (không set) → mặc định [hfd://app/*] (+ devURL/* nếu DevURL đặt).
	// Slice rỗng → chặn TẤT CẢ origin.
	AllowOrigins []string
	// CSP — Content-Security-Policy áp vào trang (inject <meta> DOCUMENT_START).
	// Ví dụ: "default-src hfd://app/; script-src hfd://app/ 'unsafe-inline'".
	// Rỗng → không đặt CSP.
	CSP string
}

// MenuItem một mục trong menu bar.
type MenuItem struct {
	Label  string
	Action string // bắn event "menu" {action} về JS khi item được chọn
}

// ipcRequest là gói tin JSON gửi từ JS runtime.
type ipcRequest struct {
	Name  string          `json:"name"`
	Args  json.RawMessage `json:"args"`
	Token string          `json:"token"` // security layer 3
}

var (
	regMu sync.RWMutex
	reg   = map[string]Handler{}

	// typedMu bảo vệ metadata type của các BindTyped — dùng cho codegen.
	typedMu      sync.Mutex
	typedEntries []typedEntry
)

// typedEntry mô tả một binding typed cho codegen TS.
type typedEntry struct {
	Name string
	In   reflect.Type
	Out  reflect.Type
}

// BindTyped đăng ký handler với kiểu dữ liệu tường minh — trái tim của codegen:
//
//	type GreetArgs struct{ Name string `json:"name"` }
//	type GreetResult struct{ Message string `json:"message"` }
//
//	func (a *App) Greet(args GreetArgs) (GreetResult, error) { … }
//
//	hfd.BindTyped("Greet", app.Greet)
//
// CLI (`hfd bindings`) đọc metadata này qua reflect để sinh bindings.d.ts
// type-safe cho phía TS. Payload JSON được unmarshal tự động vào In.
func BindTyped[In any, Out any](name string, fn func(In) (Out, error)) {
	Bind(name, func(args string) (any, error) {
		var in In
		if len(args) > 0 && args != "null" {
			if err := json.Unmarshal([]byte(args), &in); err != nil {
				return nil, fmt.Errorf("hfd: đối số của %q sai kiểu: %w", name, err)
			}
		}
		return fn(in)
	})
	typedMu.Lock()
	typedEntries = append(typedEntries, typedEntry{
		Name: name,
		In:   reflect.TypeFor[In](),
		Out:  reflect.TypeFor[Out](),
	})
	typedMu.Unlock()
}

// Bind đăng ký (hoặc ghi đè) handler cho `name`.
// Phải đăng ký trước khi Run; đổi handler lúc đang chạy vẫn hợp lệ.
func Bind(name string, h Handler) {
	regMu.Lock()
	reg[name] = h
	regMu.Unlock()
}

func lookupHandler(name string) (Handler, bool) {
	regMu.RLock()
	h, ok := reg[name]
	regMu.RUnlock()
	return h, ok
}

// backend là backend cụ thể theo nền tảng (GTK/WebKitGTK trên Linux).
type backend interface {
	run() error
	emit(name string, payload any) error
	setClipboard(text string) error
	getClipboard() (string, error)
	openFileDialog(title string, serializedFilters string, startDir string) (string, bool, error)
	saveFileDialog(title string, suggestedName string) (string, bool, error)
	showMessage(title string, message string, level int) (bool, error)
	setTitle(title string) error
	closeWindow() error
}

var (
	curMu      sync.Mutex
	curBackend backend
)

// Run mở cửa sổ + webview, nạp index.html và chạy main loop.
// Block đến khi cửa sổ bị đóng; chỉ được gọi 1 lần mỗi process.
func Run(opts Options) error {
	// Codegen mode: CLI chạy app với env HFD_CODEGEN=bindings → in schema
	// JSON ra stdout rồi thoát ngay (không mở window).
	if os.Getenv("HFD_CODEGEN") == "bindings" {
		os.Stdout.Write(buildBindingsSchema())
		return nil
	}

	if dev := os.Getenv("HFD_DEV_URL"); dev != "" && opts.DevURL == "" {
		opts.DevURL = dev
	}

	if opts.Title == "" {
		opts.Title = "HFD app"
	}
	if opts.Width == 0 {
		opts.Width = 1024
	}
	if opts.Height == 0 {
		opts.Height = 768
	}

	curMu.Lock()
	if curBackend != nil {
		curMu.Unlock()
		return errors.New("hfd: Run chỉ được gọi một lần mỗi process")
	}
	b := newEngine(opts)
	curBackend = b
	curMu.Unlock()
	return b.run()
}

// Emit bắn event Go → JS (JS bắt bằng hfd.on).
// Chạy được từ bất kỳ goroutine nào (marshal JSON trên goroutine gọi).
func Emit(name string, payload any) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: Emit trước khi Run")
	}
	return b.emit(name, payload)
}
