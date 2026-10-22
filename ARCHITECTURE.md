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
- **Phase 4 ✅ (lần đầu)**: native API linux: `OpenFileDialog/SaveFileDialog` (modal
  GTK + filter), `ShowMessage` (info/warning/error/question), `SetClipboard/
  GetClipboard`, `Notify` (notify-send), `SetTitle`, `CloseWindow`, Menu bar
  (`Options.Menu` + event `menu` → JS). Demo có card test + menu "HFD".
  Còn lại Phase 5+~7 (tray, native menu context, portal/GNotification).
- **Dev flow ✅**: `hfd dev` end-to-end trên react-demo — vite lên port → window
  nạp `http://localhost:5173` qua `HFD_DEV_URL` → IPC hoạt động; sửa App.tsx
  → vite bắn `hmr update` tự reload (không restart app).
- **Phase 6 ✅ (lần đầu)** — origin isolation 2 tầng:
  1. runtime user script inject **có allow_list** (`webkit_user_script_new`) —
     `window.hfd` chỉ tồn tại trên origin pattern khớp `Options.AllowOrigins`
     (mặc định `hfd://app/*` + `DevURL/*`); origin lạ không có bridge để gọi.
  2. mỗi IPC từ JS kèm URI của page; Go khớp pattern glob trước dispatch —
     tầng 2 phòng khi bridge nhờ lẫn. Negative test: AllowOrigins xa lạ →
     0 IPC tới Go (verified). Expression còn thiếu: CSP header, capability
     token xác thực chặt hơn glob origin.
- **Phase 3**: `@hfd/react` — hook `useHfdCall` / `useHfdEvent`; template React/Vue cho `hfd new`.
- **Phase 4**: native API thêm: dialog (open/save), clipboard, menu, notification, tray (StatusNotifier).
- **Phase 5**: backend mac (WKWebView) / win (WebView2), hỗ trợ `webkit2gtk-4.0`.
- **Phase 6**: security (allowlist handler + capability token), packaging (deb.AppImage) + signing.
