# HFD — Hyggshi Framework Desktop

An in-house desktop framework, built exactly to Hyggshi's spec (no dependency
on Wails/Tauri APIs):

| Layer | Tech | Role | Status |
|---|---|---|---|
| `core/` | **Go** | Runtime + native API (window, IPC, assets, events) | ✅ Phase 1 |
| `hfd-cli/` | **TypeScript** | CLI: `hfd dev` / `hfd build` / `hfd new` / `hfd bindings` | ✅ Phase 2 |
| frontend | **React/Vue** | Optional, UI only | ✅ Phase 3 |
| `webview_gtk.go` | **GTK3 + WebKitGTK 4.1** | Window + webview on Linux | ✅ Phase 1 |
| IPC | **postMessage (JS→Go) + evaluate_javascript (Go→JS)** | Two-way bridge | ✅ Phase 1 |
| Other native backends | WKWebView (mac), WebView2 (Win) | Later, same Go API | Phase 5 |

## Public API (Phase 1)

Go side:
```go
hfd.Bind("Greet", func(args string) (any, error) { ... })  // register handler, JSON-in
hfd.Emit("tick", map[string]any{"n": 3})                   // Go → JS event
hfd.Run(hfd.Options{Title, Width, Height, Assets, Debug})   // open window, blocks until closed
```

JS side (runtime auto-embedded `runtime/hfd.js`):
```js
const msg = await hfd.call("Greet", { name: "Hi" })        // Go returns JSON
hfd.on("tick", (data) => render(data))                      // receive Go events → UI
```

## IPC mechanics

1. **JS → Go**: JS calls
   `window.webkit.messageHandlers.hfd.postMessage(JSON.stringify({...}))`
   → WebKit fires `script-message-with-reply-received::hfd` with a reply
   object (payload is a `JSCValue`)
   → cgo wraps a Go callback, extracts the string, parses JSON, dispatches
   the `Bind`ed handler (in its own goroutine), then completes the JS promise
   via `webkit_script_message_reply_return_value` — a real awaits-the-answer
   Promise, not a synthesized resolve.
2. **Go → JS**: `webkit_web_view_evaluate_javascript("window.hfd._emit(...)")`
   (must run on the GTK main thread, marshaled via the idle pump).
3. **Assets**: served through the custom URI scheme `hfd://app/...` — read
   from the app's `embed.FS` (registered via
   `webkit_web_context_register_uri_scheme`).

## Build constraints

- Linux needs `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `libsoup-3.0-dev`
  (Ubuntu 24.04+ ships 4.1).
- Tray requires `-tags hfd_tray` (pulls in `godbus`); the default build
  keeps the core at 0 external deps.
- macOS/Windows: scaffolds in `core/webview_darwin.go`/
  `core/webview_windows.go` — they compile on the target platform but
  return a descriptive error until Phase 5 lands (needs a VM/Mac).

## Roadmap

- **Phase 1 ✅**: Linux core, two-way IPC, strings/objects, `hfd://` protocol,
  vanilla demo.
- **Phase 2 ✅**: `hfd-cli` (TypeScript ESM, 0 deps):
  - `hfd bindings` — runs `HFD_CODEGEN=bindings go run .`, collects schema JSON
    (core uses `BindTyped[In, Out]` + reflect) → generates `bindings.d.ts` +
    `bindings.js`.
  - `hfd build` — build frontend (per config) + `go build -o build/bin/`.
  - `hfd dev` — frontend dev server + `HFD_DEV_URL` env so the webview loads
    the dev URL.
  - runtime auto-injected via WebKit user script `DOCUMENT_START` (no script
    tag needed in the frontend); `Debug: true` pipes console messages to stdout.
- **Phase 3 ✅**: `packages/react` (`@hfd/react`) — hooks `useHfdCall`
  (auto/refetch), `useHfdEvent` (auto-unmount), typed `call/on`. `hfd new`
  scaffolds React + Vue apps (vite, tsconfig, demo App using the hooks).
  `examples/react-demo` builds and runs (useHfdCall calls SysInfo on mount).
- **Phase 4 ✅ (full per Linux plan)**: native API on Linux:
  `OpenFileDialog/SaveFileDialog` (modal GTK + filters), `ShowMessage`
  (info/warning/error/question), `SetClipboard/GetClipboard`, `Notify`
  (notify-send), `SetTitle`, `CloseWindow`, Menu bar (`Options.Menu` + event
  `menu` → JS), **Tray** (`SetTray` — StatusNotifierItem + DBusMenu via
  godbus, behind build tag `-tags hfd_tray`; default build keeps the core at
  0 external deps).
  Tray verified: registered with `org.kde.StatusNotifierWatcher`,
  the entry survives >8s (shell host accepts it), GetAll properties match
  the `(ia{sv}av)` layout signature. GNOME needs the AppIndicator extension
  (already installed here).
- **Dev flow ✅**: `hfd dev` end-to-end on react-demo — vite serves on its
  port → the window loads `http://localhost:5173` via `HFD_DEV_URL` → IPC
  works; editing App.tsx → vite fires `hmr update` and reloads without
  restarting the app.
- **Phase 6 ✅** — 3-layer security:
  1. capability injection: the user script injects only on origin patterns
     matching `Options.AllowOrigins` (defaults `hfd://app/*` + `DevURL/*`) —
     a foreign origin has no bridge to call (negative test: unknown
     AllowOrigins → JS error `window.hfd undefined`, 0 IPCs reach Go).
  2. URI check: every IPC carries the page URI; Go glob-matches it before
     dispatch — guards against a stray bridge living in another frame.
  3. per-launch token: `crypto/rand` 32B hex, injected together with the
     runtime (`window.__HFD_CONFIG`), every IPC must carry the correct token —
     compared with `crypto/subtle.ConstantTimeCompare`. Wrong token → reject.
  CSP: `Options.CSP` → injects `<meta http-equiv>` very early (DOCUMENT_START
  runs before app scripts). The strict-CSP demo runs without breaking IPC.
- **Phase 5 scaffold ✅**: `core/webview_windows.go` (WebView2) +
  `core/webview_darwin.go` (WKWebView) + `docs/PHASE5-BACKENDS.md`
  (per-PR architecture + 12-item test plan per backend). Cross-compile
  verified: `GOOS=windows go build` ✓, `GOOS=darwin go build` ✓ — stubs
  return a descriptive error until a VM/Mac is available.
- **Next up**: AppImage packaging, context menu, portal/GNotification.
