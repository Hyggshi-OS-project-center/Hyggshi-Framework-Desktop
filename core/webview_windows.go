//go:build windows

package hfd

import "errors"

// Scaffold Phase 5 — Windows backend using WebView2 (COM).
// A real WebView2 is NOT compiled here: needs a Windows VM to develop + test
// (docs/PHASE5-BACKENDS.md). Calling any API → returns a clear error.
const windowsPhase5Msg = "hfd: Windows backend (WebView2) — see docs/PHASE5-BACKENDS.md, not yet enabled in this version"

type windowsBackend struct{ opts Options }

func newEngine(opts Options) backend { return windowsBackend{opts} }

func (windowsBackend) run() error                    { return errors.New(windowsPhase5Msg) }
func (windowsBackend) emit(string, any) error        { return errors.New(windowsPhase5Msg) }
func (windowsBackend) setClipboard(string) error     { return errors.New(windowsPhase5Msg) }
func (windowsBackend) getClipboard() (string, error) { return "", errors.New(windowsPhase5Msg) }
func (windowsBackend) openFileDialog(string, string, string) (string, bool, error) {
	return "", false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) saveFileDialog(string, string) (string, bool, error) {
	return "", false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) showMessage(string, string, int) (bool, error) {
	return false, errors.New(windowsPhase5Msg)
}
func (windowsBackend) setTitle(string) error { return errors.New(windowsPhase5Msg) }
func (windowsBackend) closeWindow() error    { return errors.New(windowsPhase5Msg) }

// ---- WebView2 technical plan (once a VM is available) ------------------------
//
// 1. Loader: LoadLibrary("WebView2Loader.dll") →
//    CreateCoreWebView2EnvironmentWithOptions + GetCoreWebView2BrowserVersion.
// 2. Hand-rolled COM vtable calls (no wrapper): ICoreWebView2Environment →
//    CreateCoreWebView2Controller → ICoreWebView2;
//    SetVirtualHostNameToFolderMapping "app.hfd" → serving the embedded FS
//    (async ICoreWebView2_4 version; WebResourceRequested fallback to get it
//    through the network layer).
// 3. IPC: ICoreWebView2::PostWebMessageAsJson (Go→JS), an event listener
//    named WebMessageReceived (Go←JS). A shim JS compatible with hfd.js.
// 4. Promise resolution: WebMessageReceived → Go → PostWebMessageAsJson
//    {id, result} — the JS runtime keeps a map id→resolve (no reply event API needed).
// 5. Window creation: CreateWindowExW + WM_SIZE → controller.put_Bounds.
