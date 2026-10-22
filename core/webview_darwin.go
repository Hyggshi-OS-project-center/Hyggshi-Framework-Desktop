//go:build darwin

package hfd

import "errors"

// Scaffold Phase 5 — macOS backend using WKWebView (cgo Objective-C).
// A real WKWebView is NOT compiled here: needs a Mac to develop + test
// (docs/PHASE5-BACKENDS.md). Calling any API → clear error.
const darwinPhase5Msg = "hfd: macOS backend (WKWebView) — see docs/PHASE5-BACKENDS.md, not yet enabled in this version"

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

// ---- WKWebView technical plan (once a Mac is available) --------------------
//
// 1. Threading: app.Run always on the main thread (dispatch main app; Go LockOSThread).
// 2. NSApplication → NSWindow (styleMask titlebar) → contentView holding the WKWebView.
// 3. Scheme "hfd": WKWebViewConfiguration.setURLSchemeHandler(HFDSchemeHandler)
//    — implements webView(_:start:)/stopTask; serving mapping of the embedded FS like GTK.
// 4. IPC: window.webkit.messageHandlers.hfd — macOS 15+ uses
//    -addScriptMessageHandlerWithReply:name:contentWorld: (real promise reply,
//    countering WebKitGTK 2.52); older macOS: fallback defer evaluateJavaScript
//    returning {id,result} — the JS runtime keeps a map id→resolve (like WebView2).
// 5. Runtime injector: WKUserContentController.addUserScript(InjTime.documentStart).
// 6. Clipboard: NSPasteboard.general; dialogs: NSOpenPanel/NSAlert.
