# Phase 5 — Windows (WebView2) & macOS (WKWebView) backends

Status: **scaffold**. `core/webview_windows.go` + `core/webview_darwin.go`
exist and compile on their target platforms, but every method **returns a
descriptive error** — the real implementation needs a VM/Mac because we
cannot build/test cross-platform from this Linux machine.
Merge PR-by-PR as environments become available.

## Activation prerequisites per backend

| Backend | Requires | Target file |
|---|---|---|
| Windows | Windows 10+ VM/PC, Go, MSVC (to build the loader) | `core/webview_windows.go` |
| macOS | A Mac (macOS 15+ preferred for `addScriptMessageHandlerWithReply`) | `core/webview_darwin.go` |

## What stays identical across backends

- **Same engine shape**: each backend implements the `backend` interface
  (`run/emit/setClipboard/…`) — the public API layer in `hfd.go` is untouched.
- **Same runtime JS** `runtime/hfd.js`: WebKit platforms (macOS) use it as-is;
  WebView2 wraps `chrome.webview.postMessage` with a tiny shim so the bridge
  keeps the same shape. Frontend + codegen are platform-agnostic.
- **Same security model**: per-launch token + origin allowlist + CSP meta.
- **DevURL**: the webview loads the dev server URL — `hfd dev` logic is unchanged.

## Windows — WebView2

### First-run checks (onboarding)
`hfd doctor` (a new CLI command) checks: Windows 10+, Go, WebView2 Runtime
installed (registry `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-…}`)
or a fixed-version Runtime shipped alongside the app.

### Implementation steps, one PR at a time
1. **Loader + COM**: cgo calls into WebView2Loader.dll (LoadLibrary +
   `CreateCoreWebView2EnvironmentWithOptions`) — no C++ shim.
2. **Window + Controller**: `CreateWindowExW` →
   `CreateCoreWebView2Controller` → `put_Bounds` (on `WM_SIZE`).
3. **Scheme/origin**: `SetVirtualHostNameToFolderMapping` — map the virtual
   host `app.hfd` onto the embedded FS; default AllowOrigins becomes
   `https://app.hfd/*` (allowlist check still runs before dispatch).
4. **IPC**: `PostWebMessageAsJson` (Go→JS); the JS shim wraps the
   `chrome.webview` `message` event (Go←JS). Promise replies use an
   id-based map (WebView2 has no native reply event like WebKitGTK 2.52).
5. **Emit Go→JS**: `PostWebMessageAsJson` shaped `{type:"event",name,payload}`
   — the shim `_emit` demuxes to listeners.
6. **Native API mapping**: clipboard (CF_UNICODETEXT), dialogs
   (IFileOpenDialog + COMDLG_FILTERSPEC), Notify (Shell_NotifyIconW/Win toast),
   SetTitle (SetWindowTextW), CloseWindow (WM_CLOSE), menu (CreateMenu).

### Windows test plan (rerun on the VM for every PR)
```
A1  window opens with correct Options.Title; resize keeps webview bounds right
A2  reload re-injects the runtime at DOCUMENT_START
A3  IPC roundtrip: BindEcho → JS call → Go → promise resolves with the value
A4  emit Go→JS: 5 ticks counted correctly in a JS listener
A5  origin isolation: navigate to data:text/html → window.hfd is absent
A6  token: tampering with the token from the console → call rejected
A7  scheme: <img src="https://app.hfd/img.png"> loads from the embedded FS
A8  clipboard: Set/Get across other apps
A9  OpenFileDialog returns a real path; Cancel → canceled=true
A10 dialog filter lists only allowed extensions
A11 Notify (toast) shows; clicking it does not crash the app
A12 closing the window → Run() returns nil, no zombie process
```

## macOS — WKWebView

### Implementation steps
1. **Lifecycle**: NSApplication on the main thread (Go `LockOSThread`
   main); ObjC callbacks dispatch back to a Go goroutine (same pump pattern).
2. **Window + WKWebView**: NSWindow (FullSizeContentView) + WKWebView;
   `userContentController.addUserScript` as a documentStart user script.
3. **The "hfd" scheme**: `setURLSchemeHandler` (WKURLSchemeHandler) —
   serves the embedded FS with the same mime table as the GTK backend.
4. **IPC**: on macOS 15+ use `addScriptMessageHandlerWithReply`
   (true promise-reply, same as the GTK backend); older macOS falls back to
   id-based replies via `evaluateJavaScript`. Frontend unchanged.
5. **Emit Go→JS**: `evaluateJavaScript` runs `window.hfd._emit(…)`.
6. **Native API mapping**: clipboard NSPasteboard; dialogs NSOpenPanel/
   NSSavePanel; Notify UNUserNotificationCenter; SetTitle window.title;
   menu NSMenu (title bar).

### macOS test plan (mirrors the A1–A12 items, differing at the scheme)
```
B1  window opens full-size contentView; retina does not double-scale
B2  runtime injects before app scripts (documentStart)
B3  IPC roundtrip with a real reply handler (macOS 15+); id-based fallback on older
B4  emit: "tick"/"tray" events with correct payloads
B5  origin: page URI matches the allowlist before dispatch
B6  wrong token → rejected
B7  scheme hfd:// returns the right mime for CSS/JS/PNG
B8  Pasteboard Set/Get (including text from other apps)
B9  NSOpenPanel stays modal without blocking the main thread (completion handler)
B10 Save panel + Notify (Info.plist requirements apply when bundling)
B11 NSApplication terminate → Go observes exit 0
B12 Retain/release of ObjC blocks — no leaks across repeated callbacks
```

## PR ladder per backend

| PR | Windows | macOS |
|----|---------|-------|
| 1  | Loader + window + blank webview | NSApp + NSWindow + WKWebView boot |
| 2  | VirtualHost → embedded FS + runtime inject | URLSchemeHandler + user script |
| 3  | IPC roundtrip + emit | IPC roundtrip (15+ reply handler / id-based fallback) |
| 4  | Full native API set + running demo | Full native API set + running demo |
| 5  | Tray: Shell_NotifyIconW | Tray: NSStatusItem |

Every PR must include logs/output that prove each test item — nothing
passes on trust.
