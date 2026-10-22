//go:build darwin

package hfd

import "errors"

// Scaffold Phase 5 — backend macOS bằng WKWebView (cgo Objective-C).
// KHÔNG compile WKWebView thật ở đây: cần Mac để phát triển + test
// (docs/PHASE5-BACKENDS.md). Gọi API nào → lỗi rõ ràng.
const darwinPhase5Msg = "hfd: backend macOS (WKWebView) — xem docs/PHASE5-BACKENDS.md, chưa bật trong bản này"

type darwinBackend struct{ opts Options }

func newEngine(opts Options) backend { return darwinBackend{opts} }

func (darwinBackend) run() error                    { return errors.New(darwinPhase5Msg) }
func (darwinBackend) emit(string, any) error        { return errors.New(darwinPhase5Msg) }
func (darwinBackend) setClipboard(string) error     { return errors.New(darwinPhase5Msg) }
func (darwinBackend) getClipboard() (string, error) { return "", errors.New(darwinPhase5Msg) }
func (darwinBackend) openFileDialog(string, string, string) (string, bool, error) {
	return "", false, errors.New(darwinPhase5Msg)
}
func (darwinBackend) saveFileDialog(string, string) (string, bool, error) {
	return "", false, errors.New(darwinPhase5Msg)
}
func (darwinBackend) showMessage(string, string, int) (bool, error) {
	return false, errors.New(darwinPhase5Msg)
}
func (darwinBackend) setTitle(string) error { return errors.New(darwinPhase5Msg) }
func (darwinBackend) closeWindow() error    { return errors.New(darwinPhase5Msg) }

// ---- Kế hoạch kỹ thuật WKWebView (khi có Mac) --------------------------------
//
// 1. Threading: app.Run luôn trên main thread (dispatch main app; Go LockOSThread).
// 2. NSApplication → NSWindow (styleMask titlebar) → contentView chứa WKWebView.
// 3. Scheme "hfd": WKWebViewConfiguration.setURLSchemeHandler(HFDSchemeHandler)
//    — implements webView(_:start:)/stopTask; serving mapping embed FS như GTK.
// 4. IPC: window.webkit.messageHandlers.hfd — macOS 15+ dùng
//    -addScriptMessageHandlerWithReply:name:contentWorld: (promise reply thật,
//    đối đằng WebKitGTK 2.52); macOS cũ: fallback defer evaluateJavaScript
//    trả {id,result} — runtime JS giữ map id→resolve (giống WebView2).
// 5. Injector runtime: WKUserContentController.addUserScript(InjTime.documentStart).
// 6. Clipboard: NSPasteboard.general; dialog: NSOpenPanel/NSAlert.
