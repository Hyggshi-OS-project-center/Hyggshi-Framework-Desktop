# Phase 5 — Backend Windows (WebView2) & macOS (WKWebView)

Trạng thái: **scaffold**. `core/webview_windows.go` + `core/webview_darwin.go`
đã tồn tại, compile được trên đúng nền tảng, nhưng **trả lỗi khi gọi** —
việc triển khai thật cần VM/Mac vì không build/test chéo được từ Linux.
Merge theo từng PR khi có môi trường.

## Điều kiện kích hoạt mỗi backend

| Backend  | Cần                                                        | File đích                    |
|----------|-------------------------------------------------------------|------------------------------|
| Windows  | VM/PC Windows 10+ + Go + MSVC (biên dịch loader)            | `core/webview_windows.go`    |
| macOS    | Mac, ưu tiên macOS 15+ (có `addScriptMessageHandlerWithReply`) | `core/webview_darwin.go`  |

## Kiến trúc chung giữ nguyên giữa các backend

- **Cùng shape engine**: backend implement đủ `backend` interface
  (`run/emit/setClipboard/…`) — tầng public API trong `hfd.go` không đổi.
- **Cùng runtime JS** `runtime/hfd.js`: WebKit family (macOS) dùng nguyên
  trạng; WebView2 dùng shim nhỏ bọc `chrome.webview.postMessage` thành
  bridge cùng shape. Frontend + codegen không phân biệt nền tảng.
- **Cùng security model**: per-launch token + allowlist origin + CSP meta.
- **DevURL**: webview nạp địa chỉ dev server — `hfd dev` giữ nguyên logic.

## Windows — WebView2

### Kiểm đầu (khi onboarding dev)
`hfd doctor` (lệnh mới) kiểm: Windows 10+, Go, WebView2 Runtime đã cài
(registry `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-…}`)
hoặc đóng gói fixed-version Runtime cùng app.

### Sơ đồ triển khai từng PR
1. **Loader + COM**: cgo gọi WebView2Loader.dll (LoadLibrary +
   `CreateCoreWebView2EnvironmentWithOptions`) — không C++ shim.
2. **Window + Controller**: `CreateWindowExW` →
   `CreateCoreWebView2Controller` → `put_Bounds` (theo `WM_SIZE`).
3. **Scheme/origin**: `SetVirtualHostNameToFolderMapping` — virtual host
   `app.hfd` map vào embed FS; AllowOrigins mặc định đổi thành `https://app.hfd/*`
   (allowlist vẫn hoạt động trước khi dispatch).
4. **IPC**: `PostWebMessageAsJson` (Go→JS); JS shim bọc listener
   `chrome.webview` event `message` (Go←JS). Promise reply dùng id-based
   map (WebView2 không có native reply event như WebKitGTK 2.52).
5. **Emit Go→JS**: `PostWebMessageAsJson` dạng `{type:"event",name,payload}`
   — shim `_emit` nhận và dispatch cho listener.
6. **Native API mapping**: clipboard (CF_UNICODETEXT), dialog
   (IFileOpenDialog + COMDLG_FILTERSPEC), Notify (Shell_NotifyIconW/Win toast),
   SetTitle (SetWindowTextW), CloseWindow (WM_CLOSE), menu (CreateMenu).

### Test plan Windows (chạy lại mỗi PR trên VM)
```
A1  window mở đúng Options.Title; resize → webview bounds theo
A2  reload: runtime inject lại đúng DOCUMENT_START
A3  IPC roundtrip: BindEcho → JS call → Go → promise trả đúng giá trị
A4  emit Go→JS: 5 tick đếm đủ trong listener JS
A5  origin isolation: navigate tới data:text/html → window.hfd KHÔNG xuất hiện
A6  token: đổi token bằng console → call bị từ chối
A7  scheme: <img src="https://app.hfd/img.png"> nạp từ embed FS
A8  clipboard: Set/Get qua các app khác
A9  OpenFileDialog trả path thật; Cancel → canceled=true
A10 filter dialog chỉ hiện đúng đuôi file
A11 Notify (toast) hiện đúng, app không crash khi click
A12 đóng window → Run() return nil, không còn process zombie
```

## macOS — WKWebView

### Sơ đồ triển khai
1. **Lifecycle**: NSApplication trên main thread (Go `LockOSThread` main);
   callback ObjC → dispatch về goroutine Go tương đương pump hiện tại.
2. **Window + WKWebView**: NSWindow (FullSizeContentView) + WKWebView;
   `userContentController.addUserScript` runtime (documentStart).
3. **Scheme "hfd"**: `setURLSchemeHandler` (WKURLSchemeHandler) — serving
   embed FS + mime map giống GTK backend.
4. **IPC**: macOS 15+ dùng `addScriptMessageHandlerWithReply`
   (promise-reply thật, tương đương GTK backend); macOS cũ rơi về id-based
   reply qua `evaluateJavaScript`. Frontend không đổi.
5. **Emit Go→JS**: `evaluateJavaScript` chạy `window.hfd._emit(…)`.
6. **Native API mapping**: clipboard NSPasteboard; dialog NSOpenPanel/
   NSSavePanel; Notify UNUserNotificationCenter; SetTitle window.title;
   menu NSMenu (title bar).

### Test plan macOS (tương tự A1–A12, khác điểm scheme)
```
B1  window mở full-size contentView; retina không double-scale
B2  runtime inject theo documentStart (nạp trước script app)
B3  IPC roundtrip bằng reply handler thật (macOS 15+); fallback id-based chạy trên cũ
B4  emit: event "tick"/"tray" đúng payload
B5  origin: URI page khớp allowlist trước dispatch
B6  token sai → từ chối
B7  scheme hfd:// trả đúng mime cho CSS/JS/PNG
B8  Pasteboard Set/Get (kể cả text từ app khác)
B9  NSOpenPanel modal không block main thread (completion handler)
B10 Save panel + Notify (cần Info.plist phù hợp khi đóng gói bundle)
B11 NSApplication terminate → Go nhận exit 0
B12 Retain/release các block ObjC — không leak qua callback nhiều lần
```

## Lộ trình PR

| PR | Windows                                          | macOS                                            |
|----|--------------------------------------------------|--------------------------------------------------|
| 1  | Loader + window + webview trắng                  | NSApp + NSWindow + WKWebView khởi động           |
| 2  | VirtualHost → embed FS + runtime inject          | URLSchemeHandler + user script                   |
| 3  | IPC roundtrip + emit                             | IPC roundtrip (reply 15+ / id-based fallback)    |
| 4  | Native API đầy đủ + demo chạy                    | Native API đầy đủ + demo chạy                    |
| 5  | Tray: Shell_NotifyIconW                          | Tray: NSStatusItem                               |

Mỗi PR phải có log/output chứng minh từng mục test — không pass trên niềm tin.
