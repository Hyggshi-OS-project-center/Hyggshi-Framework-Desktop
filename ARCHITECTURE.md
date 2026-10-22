# HFD — Hyggshi Framework Desktop

Framework desktop riêng, theo đúng spec của Hyggshi (không phụ thuộc API của Wails / Tauri):

| Lớp | Công nghệ | Vai trò | Trạng thái |
|---|---|---|---|
| `core/` | **Go** | Runtime + native API (cửa sổ, IPC, assets, events) | ✅ Phase 1 |
| `hfd-cli/` (chưa dựng) | **TypeScript** | CLI: `hfd dev` / `hfd build` / `hfd new` / `hfd bindings` | Phase 2 |
| frontend | **React/Vue** | Tùy chọn, chỉ phục vụ giao diện | Phase 3 |
| `webview_gtk.go` | **GTK3 + WebKitGTK 4.1** | Cửa sổ + webview trên Linux | ✅ Phase 1 |
| IPC | **postMessage (JS→Go) + evaluate_javascript (Go→JS)** | Cầu nối 2 chiều | ✅ Phase 1 |
| Backend native khác | WKWebView (mac), WebView2 (Win) | Làm sau, cùng API Go | Phase 5 |

## API công khai (Phase 1)

Go side:
```go
hfd.Bind("Greet", func(args string) (any, error) { ... })  // đăng ký handler, nhận JSON-in
hfd.Emit("tick", map[string]any{"n": 3})                   // Go → JS event
hfd.Run(hfd.Options{Title, Width, Height, Assets, Debug})   // mở cửa sổ, block đến khi đóng
```

JS side (runtime nhúng sẵn `__runtime/hfd.js`):
```js
const msg = await hfd.call("Greet", { name: "Hi" })        // Go → trả JSON
hfd.on("tick", (data) => render(data))                      // bắt event Go → UI
```

## Cơ chế IPC

1. **JS → Go**: JS gọi `window.webkit.messageHandlers.ipc.postMessage(JSON.stringify({...}))`
   → WebKit bắn tín hiệu `script-message-received::ipc` (payload là `JSCValue`)
   → cgo bọc Go callback, nhận string, parse JSON, dispatch handler đã `Bind` (chạy trong goroutine riêng).
2. **Go → JS**: handler xử lý song song, trả về → Marshal JSON
   → dùng `webkit_web_view_evaluate_javascript("window.hfd._resolve(<id>, <json>)")` (chỉ chạy trên GTK main thread, marshal qua idle pump).
3. **Assets**: cái thông qua custom URI scheme `hfd://app/...` — đọc từ `embed.FS` của app
   (cùng mức như Wails nhưng do ta đăng ký handler bằng `webkit_web_context_register_uri_scheme`).

## Constraints biên dịch

- Linux cần `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `libsoup-3.0-dev` (Ubuntu 24.04+ đều dùng 4.1).
- Thêm tag cũ 4.0 sau (Phase 5) qua build tag.
- macOS/Windows: stub `unsupported.go` cho tới khi hỗ trợ.

## Roadmap

- **Phase 1 ✅**: Linux core, IPC 2 chiều, strings/objects, protocol `hfd://`, demo vanilla.
- **Phase 2 ✅**: `hfd-cli` (TypeScript ESM, 0 dep):
  - `hfd bindings` — chạy `HFD_CODEGEN=bindings go run .` thu schema JSON (core dùng
    `BindTyped[In, Out]` + reflect) → sinh `bindings.d.ts` + `bindings.js`.
  - `hfd build` — build frontend (tùy cấu hình) + `go build -o build/bin/`.
  - `hfd dev` — frontend dev server + env `HFD_DEV_URL` cho webview nạp dev URL.
  - runtime inject tự động qua WebKit user script `DOCUMENT_START` (frontend không
    cần script tag); `Debug: true` bật console messages → stdout.
- **Phase 3 ✅**: `packages/react` (`@hfd/react`) — hook `useHfdCall(auto/refetch)`,
  `useHfdEvent(auto-unmount)`, `call/on` typed; tsc clean. `hfd new` scaffold
  react + vue (vite, tsconfig, App demo dùng hooks). Demo `examples/react-demo`
  đã build + chạy (useHfdCall gọi SysInfo ngay khi mount).
- **Phase 4 ✅ (đầy đủ theo plan linux-hop)**: native API linux:
  `OpenFileDialog/SaveFileDialog` (modal GTK + filter), `ShowMessage`
  (info/warning/error/question), `SetClipboard/GetClipboard`, `Notify`
  (notify-send), `SetTitle`, `CloseWindow`, Menu bar (`Options.Menu` + event
  `menu` → JS), **Tray** (`SetTray` — StatusNotifierItem + DBusMenu qua
  godbus, sau build tag `-tags hfd_tray`; core mặc định vẫn 0 external dep).
  Tray verified: đăng ký vào `org.kde.StatusNotifierWatcher` thành công,
  object tiếp tục giữ >8s (shell host chấp nhận), GetAll properties chuẩn
  signature `(ia{sv}av)` của layout. GNOME cần extension AppIndicator (đã có).
- **Dev flow ✅**: `hfd dev` end-to-end trên react-demo — vite lên port → window
  nạp `http://localhost:5173` qua `HFD_DEV_URL` → IPC hoạt động; sửa App.tsx
  → vite bắn `hmr update` tự reload (không restart app).
- **Phase 6 ✅** — security 3 tầng:
  1. capability inject: user script chỉ inject trên origin pattern khớp
     `Options.AllowOrigins` (mặc định `hfd://app/*` + `DevURL/*`) — origin lạ
     không có bridge nào để gọi (negative test: AllowOrigins xa lạ → JS lỗi
     `window.hfd undefined`, 0 IPC tới Go).
  2. URI check: mỗi IPC kèm URI page; Go khớp glob (constant-time không,
     glob) trước dispatch — phòng bridge nhờ lẫn qua frame.
  3. per-launch token: `crypto/rand` 32B hex, inject cùng runtime
     (`window.__HFD_CONFIG`), mọi IPC phải mang token đúng — compare bằng
     `crypto/subtle.ConstantTimeCompare`. Token khác → từ chối.
  CSP: `Options.CSP` → inject `<meta http-equiv>` rất sớm (DOCUMENT_START
  chạy trước script app). CSP strict demo đã chạy không phá IPC.
- **Phase 5 scaffold ✅**: `core/webview_windows.go` (WebView2) +
  `core/webview_darwin.go` (WKWebView) + `docs/PHASE5-BACKENDS.md`
  (kiến trúc per-PR + 12 mục test plan/backend). Cross-compile verified:
  `GOOS=windows go build` ✓, `GOOS=darwin go build` ✓ — chưa chạy được
  mà chỉ stub trả lỗi có hướng dẫn; gọi cụm khi có VM/Mac.
- **Phase 3**: `@hfd/react` — hook `useHfdCall` / `useHfdEvent`; template React/Vue cho `hfd new`.
- **Phase 5**: backend mac (WKWebView) / win (WebView2) — scaffold sẵn, awaiting VM.
- **Tiếp theo**: packaging (deb/AppImage), menu context, portal/GNotification, `hfd doctor`.
