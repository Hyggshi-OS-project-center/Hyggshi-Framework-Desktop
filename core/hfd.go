// Package hfd — Go runtime of Hyggshi Framework Desktop (HFD).
//
// Architecture per the Hyggshi spec:
//   - Go:              runtime + native API (this package).
//   - TypeScript:      CLI/devtools (phase 2).
//   - React and Vue:   optional frontends.
//   - GTK + WebKitGTK: windows/webview on Linux.
//   - IPC/bindings:    script message handler with native reply (JS→Go)
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

// Handler runs when JS calls hfd.call(name, args).
// `args` is the raw JSON of the payload (the string "null" when JS passes nothing).
// The return value is run through json.Marshal and handed back to JS; an error
// makes the promise on the JS side reject with that error message.
type Handler func(args string) (any, error)

// Options configures the app.
type Options struct {
	// Window title. Default "HFD app".
	Title string
	// Window size. Default 1024×768.
	Width  int
	Height int
	// Assets holds the frontend build (the root must contain index.html).
	// Use fs.Sub(embedFS, "frontend/dist") if the embed has a prefix.
	// Nil → serves only the runtime JS + a default index page.
	Assets fs.FS
	// Debug enables the WebKit Inspector (right-click → Inspect Element).
	Debug bool
	// DevURL for dev mode (e.g. "http://localhost:5173" — vite dev server).
	// If empty, the CLI may set it via the HFD_DEV_URL env var.
	DevURL string
	// GTK menu bar (linux) — clicking an item → JS receives the "menu" event {action}.
	MenuLabel string // default "Menu"
	Menu      []MenuItem
	// AllowOrigins — URI patterns allowed to run the bridge IPC `window.hfd`.
	// Glob patterns where '*' matches any string: "hfd://app/*", "http://localhost:5173/*".
	// Nil (unset) → default [hfd://app/*] (+ devURL/* if DevURL is set).
	// Empty slice → block ALL origins.
	AllowOrigins []string
	// CSP — Content-Security-Policy applied to the page (injecting <meta> at DOCUMENT_START).
	// Example: "default-src hfd://app/; script-src hfd://app/ 'unsafe-inline'".
	// Empty → no CSP is set.
	CSP string
}

// MenuItem is one item in the menu bar.
type MenuItem struct {
	Label  string
	Action string // emits the "menu" event {action} back to JS when the item is selected
}

// ipcRequest is the JSON packet sent from the JS runtime.
type ipcRequest struct {
	Name  string          `json:"name"`
	Args  json.RawMessage `json:"args"`
	Token string          `json:"token"` // security layer 3
}

var (
	regMu sync.RWMutex
	reg   = map[string]Handler{}

	// typedMu protects the type metadata of the BindTyped entries — used by codegen.
	typedMu      sync.Mutex
	typedEntries []typedEntry
)

// typedEntry describes one typed binding for TS codegen.
type typedEntry struct {
	Name string
	In   reflect.Type
	Out  reflect.Type
}

// BindTyped registers a handler with explicit data types — the heart of the codegen:
//
//	type GreetArgs struct{ Name string `json:"name"` }
//	type GreetResult struct{ Message string `json:"message"` }
//
//	func (a *App) Greet(args GreetArgs) (GreetResult, error) { … }
//
//	hfd.BindTyped("Greet", app.Greet)
//
// The CLI (`hfd bindings`) reads this metadata via reflect to generate
// type-safe bindings.d.ts for the TS side. The JSON payload is unmarshaled
// automatically into In.
func BindTyped[In any, Out any](name string, fn func(In) (Out, error)) {
	Bind(name, func(args string) (any, error) {
		var in In
		if len(args) > 0 && args != "null" {
			if err := json.Unmarshal([]byte(args), &in); err != nil {
				return nil, fmt.Errorf("hfd: argument of %q has the wrong type: %w", name, err)
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

// Bind registers (or overwrites) the handler for `name`.
// Must be registered before Run; changing the handler while running is still valid.
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

// backend is the platform-specific backend (GTK/WebKitGTK on Linux).
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

// Run opens the window + webview, loads index.html, and runs the main loop.
// Blocks until the window is closed; may only be called once per process.
func Run(opts Options) error {
	// Codegen mode: the CLI runs the app with env HFD_CODEGEN=bindings → prints
	// the JSON schema to stdout and exits immediately (no window opened).
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
		return errors.New("hfd: Run may only be called once per process")
	}
	b := newEngine(opts)
	curBackend = b
	curMu.Unlock()
	return b.run()
}

// Emit sends a Go → JS event (JS listens for it with hfd.on).
// Usable from any goroutine (JSON marshaling happens on the calling goroutine).
func Emit(name string, payload any) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: Emit before Run")
	}
	return b.emit(name, payload)
}
