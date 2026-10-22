//go:build windows

package hfd

import "errors"

// Scaffold Phase 5 — backend Windows bằng WebView2 (COM).
// KHÔNG compile WebView2 thật ở đây: cần VM Windows để phát triển + test
// (docs/PHASE5-BACKENDS.md). Gọi bất kỳ API nào → trả lỗi rõ ràng.
const windowsPhase5Msg = "hfd: backend Windows (WebView2) — xem docs/PHASE5-BACKENDS.md, chưa bật trong bản này"

type windowsBackend struct{ opts Options }

func newEngine(opts Options) backend { return windowsBackend{opts} }

func (windowsBackend) run() error                          { return errors.New(windowsPhase5Msg) }
func (windowsBackend) emit(string, any) error              { return errors.New(windowsPhase5Msg) }
func (windowsBackend) setClipboard(string) error           { return errors.New(windowsPhase5Msg) }
func (windowsBackend) getClipboard() (string, error)       { return "", errors.New(windowsPhase5Msg) }
func (windowsBackend) openFileDialog(string, string, string) (string, bool, error) {
	return "", false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) saveFileDialog(string, string) (string, bool, error) {
	return "", false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) showMessage(string, string, int) (bool, error) {
	return false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) setTitle(string) error  { return errors.New(windowsPhase5Msg) }
func (windowsBackend) closeWindow() error     { return errors.New(windowsPhase5Msg) }

// ---- Kế hoạch kỹ thuật WebView2 (khi có VM) ---------------------------------
//
// 1. Loader: LoadLibrary("WebView2Loader.dll") →
//    CreateCoreWebView2EnvironmentWithOptions + GetCoreWebView2BrowserVersion.
// 2. COM vtable gọi tay (không wrapper): ICoreWebView2Environment →
//    CreateCoreWebView2Controller → ICoreWebView2;
//    SetVirtualHostNameToFolderMapping "app.hfd" → serving embed FS
//    (async ICoreWebView2_4 version; fallback WebResourceRequested để net đi bản).
// 3. IPC: ICoreWebView2::PostWebMessageAsJson (Go→JS), ClipBrowserEventHandler
//    listener tên WebMessageReceived (Go←JS). shim JS tương thích hfd.js.
// 4. Kết quả promise: WebMessageReceived → Go → PostWebMessageAsJson
//    {id, result} — runtime JS giữ map id→resolve (không cần reply event API).
// 5. Owin creation: CreateWindowExW + WM_SIZE → controller.put_Bounds.
