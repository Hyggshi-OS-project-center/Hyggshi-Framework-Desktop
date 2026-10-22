//go:build linux

package hfd

/*
#cgo pkg-config: webkit2gtk-4.1

#include <gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <jsc/jsc.h>
#include <stdlib.h>
#include <stdint.h>

extern void hfdGoOnScriptMessage(char *payload, char *uri, void *jscCtx, void *reply);
extern void hfdGoOnSchemeRequest(void *request);
extern void hfdGoOnPump(void);
extern void hfdGoOnWindowClosed(void);
extern void hfdGoOnMenu(char *action);

static void hfdApplyMenu(GtkWidget *box, const char *serialized); // fwd

static GtkWidget                *hfdWin  = NULL;
static WebKitWebView            *hfdView = NULL;
static WebKitUserContentManager *hfdUcm  = NULL;
static GtkWidget                *hfdBox  = NULL;   // vbox chứa menu + webview
static GtkWidget                *hfdMenuBar = NULL; // thay thế được bằng SetMenu

static void hfdWindowDestroy(GtkWidget *w, gpointer data) {
	hfdGoOnWindowClosed();
	gtk_main_quit();
}

static void hfdSchemeRequestCallback(WebKitURISchemeRequest *request, gpointer data) {
	hfdGoOnSchemeRequest((void *)request);
}

// "a\nb\nc" → gchar** NULL-terminated (caller g_strfreev).
static gchar **hfdParseLines(const char *joined) {
	GPtrArray *arr = g_ptr_array_new_with_free_func(g_free);
	const char *p = joined;
	while (p != NULL && *p != '\0') {
		const char *nl = strchr(p, '\n');
		size_t len = nl ? (size_t)(nl - p) : strlen(p);
		if (len > 0) {
			g_ptr_array_add(arr, g_strndup(p, len));
		}
		p = nl ? nl + 1 : NULL;
	}
	g_ptr_array_add(arr, NULL);
	return (gchar **)g_ptr_array_free(arr, FALSE);
}

// Script nguồn runtime + allowlist origin (window.hfd chỉ tồn tại ở origin
// được phép — security model của HFD: bridge = capability).
static WebKitUserScript *hfdNewRuntimeScript(const char *src, const char *patternsJoined) {
	gchar **allowList = hfdParseLines(patternsJoined);
	WebKitUserScript *rs = webkit_user_script_new(
		src,
		WEBKIT_USER_CONTENT_INJECT_ALL_FRAMES,
		WEBKIT_USER_SCRIPT_INJECT_AT_DOCUMENT_START,
		(const gchar *const *)allowList,
		NULL);
	g_strfreev(allowList);
	return rs;
}

// script-message-with-reply-received::hfd — WebKit giữ promise JS; sống lâu hơn
// callback vì ta ref reply + JSCContext, rồi trả giá trị khi handler Go xong.
// Chú ý: signal này truyền JSCValue thẳng (không phải WebKitJavascriptResult).
static gboolean hfdScriptMessageCallback(WebKitUserContentManager *manager,
                                         JSCValue *value,
                                         WebKitScriptMessageReply *reply,
                                         gpointer user) {
	JSCContext *ctx     = jsc_value_get_context(value);
	char       *payload = jsc_value_to_string(value); // copy — bên Go g_free
	gchar      *uri     = g_strdup(webkit_web_view_get_uri(hfdView)); // page hiện tại

	webkit_script_message_reply_ref(reply);
	g_object_ref((GObject *)ctx);
	hfdGoOnScriptMessage(payload, uri, (void *)ctx, (void *)reply);
	return TRUE;
}

static gboolean hfdPumpCallback(gpointer data) {
	hfdGoOnPump();
	return G_SOURCE_CONTINUE;
}

static WebKitWebView *hfdBootstrap(int width, int height, const char *title, int devtools,
                                   const char *runtimeSrc, const char *devURL,
                                   const char *menuSerialized, const char *allowListJoined) {
	gtk_init(NULL, NULL);

	hfdWin = gtk_window_new(GTK_WINDOW_TOPLEVEL);
	gtk_window_set_title(GTK_WINDOW(hfdWin), title);
	gtk_window_set_default_size(GTK_WINDOW(hfdWin), width, height);
	gtk_window_set_position(GTK_WINDOW(hfdWin), GTK_WIN_POS_CENTER);

	hfdView = WEBKIT_WEB_VIEW(webkit_web_view_new());
	hfdUcm  = webkit_web_view_get_user_content_manager(hfdView);

	if (devtools) {
		webkit_settings_set_enable_developer_extras(webkit_web_view_get_settings(hfdView), TRUE);
		// console.log/error của page in ra stdout/stderr — cần để debug dev.
		webkit_settings_set_enable_write_console_messages_to_stdout(
			webkit_web_view_get_settings(hfdView), TRUE);
	}

	// IPC JS→Go: postMessage trả về Promise thật; Go hoàn tất bằng hfdReplyJSON/hfdReplyError.
	webkit_user_content_manager_register_script_message_handler_with_reply(hfdUcm, "hfd", NULL);
	g_signal_connect(hfdUcm, "script-message-with-reply-received::hfd",
	                 G_CALLBACK(hfdScriptMessageCallback), NULL);

	// Inject runtime hfd.js tự động (DOCUMENT_START) — frontend không cần
	// <script src> thủ công; hoạt động cả với Assets lẫn DevURL (dev server).
	// Allow list giới hạn origin được inject → bridge = capability.
	if (runtimeSrc != NULL) {
		WebKitUserScript *rs = hfdNewRuntimeScript(runtimeSrc, allowListJoined);
		webkit_user_content_manager_add_script(hfdUcm, rs);
		webkit_user_script_unref(rs);
	}

	// Serve assets qua custom scheme hfd://app/...
	webkit_web_context_register_uri_scheme(webkit_web_view_get_context(hfdView),
	                                       "hfd", hfdSchemeRequestCallback, NULL, NULL);

	hfdBox = gtk_box_new(GTK_ORIENTATION_VERTICAL, 0);
	gtk_container_add(GTK_CONTAINER(hfdWin), hfdBox);

	// Menu bar (nếu Options.Menu được cấu hình).
	if (menuSerialized != NULL && menuSerialized[0] != '\0') {
		hfdApplyMenu(hfdBox, menuSerialized);
	}

	gtk_box_pack_start(GTK_BOX(hfdBox), GTK_WIDGET(hfdView), TRUE, TRUE, 0);
	gtk_widget_show_all(GTK_WIDGET(hfdWin));

	if (devURL != NULL && devURL[0] != '\0') {
		// Dev mode: nạp dev server (vite HMR...) thay vì scheme.
		webkit_web_view_load_uri(hfdView, devURL);
	} else {
		webkit_web_view_load_uri(hfdView, "hfd://app/index.html");
	}

	g_signal_connect(hfdWin, "destroy", G_CALLBACK(hfdWindowDestroy), NULL);
	g_timeout_add(10, hfdPumpCallback, NULL);

	return hfdView;
}

// Chỉ được gọi trong GTK main thread (pump đảm nhiệm).
static void hfdEvaluate(void *view, const char *script, gssize length) {
	webkit_web_view_evaluate_javascript(WEBKIT_WEB_VIEW(view), script, length,
	                                    NULL, NULL, NULL, NULL, NULL);
}

static void hfdFinishRequest(void *request, void *data, gssize length, const char *mime) {
	GInputStream *stream;
	if (length > 0) {
		gchar *copy = g_memdup2(data, length);
		stream = g_memory_input_stream_new_from_data(copy, length, (GDestroyNotify)g_free);
	} else {
		stream = g_memory_input_stream_new();
	}
	webkit_uri_scheme_request_finish(WEBKIT_URI_SCHEME_REQUEST(request), stream, length, mime);
}

static const char *hfdRequestGetURI(void *request) {
	return webkit_uri_scheme_request_get_uri(WEBKIT_URI_SCHEME_REQUEST(request));
}

// Trả kết primitive JSON → promise phía JS. Chỉ gọi trên main thread.
static void hfdReplyJSON(void *reply, void *jscCtx, const char *json) {
	JSCContext *ctx = JSC_CONTEXT(jscCtx);
	JSCValue   *v   = jsc_value_new_from_json(ctx, json);
	if (v != NULL) {
		webkit_script_message_reply_return_value(reply, v);
	} else {
		webkit_script_message_reply_return_error_message(reply, "hfd: invalid json reply");
	}
	webkit_script_message_reply_unref(((WebKitScriptMessageReply *)reply));
	g_object_unref((GObject *)ctx);
}

static void hfdReplyError(void *reply, void *jscCtx, const char *message) {
	webkit_script_message_reply_return_error_message(reply, message);
	webkit_script_message_reply_unref(((WebKitScriptMessageReply *)reply));
	g_object_unref((GObject *)jscCtx);
}

// Thu hồi ref nếu handler quá muộn (app đã đóng) — không gửi gì.
static void hfdReleaseScriptReply(void *reply, void *jscCtx) {
	webkit_script_message_reply_unref(((WebKitScriptMessageReply *)reply));
	g_object_unref((GObject *)jscCtx);
}

static void hfdCloseWindow(void) {
	if (hfdWin != NULL) {
		gtk_widget_destroy(hfdWin);
	}
}

// ---- menu bar --------------------------------------------------------------
// Format serialized: "MenuLabel\n" + mỗi item "Label\x1fAction\n"
extern void hfdGoOnMenu(char *action); // Go export nhận char*

static void hfdMenuItemActivate(GtkMenuItem *self, gpointer data) {
	hfdGoOnMenu((char *)data); // data là strdup — closure sẽ g_free
}

// Tạo menu bar từ chuỗi serialize và chèn lên đầu vbox.
static void hfdApplyMenu(GtkWidget *box, const char *serialized) {
	// phá menu bar cũ nếu có
	if (hfdMenuBar != NULL) {
		gtk_widget_destroy(hfdMenuBar);
		hfdMenuBar = NULL;
	}
	if (serialized == NULL || serialized[0] == '\0') {
		return;
	}

	const char *nl = strchr(serialized, '\n');
	size_t labelLen = nl ? (size_t)(nl - serialized) : strlen(serialized);
	gchar *menuLabel = g_strndup(serialized, labelLen);

	GtkWidget *bar    = gtk_menu_bar_new();
	GtkWidget *parent = gtk_menu_item_new_with_label(menuLabel);
	GtkWidget *sub    = gtk_menu_new();
	gtk_menu_item_set_submenu(GTK_MENU_ITEM(parent), sub);
	gtk_menu_shell_append(GTK_MENU_SHELL(bar), parent);

	const char *p = (nl != NULL) ? nl + 1 : NULL;
	while (p != NULL && *p != '\0') {
		const char *next = strchr(p, '\n');
		size_t lineLen = next ? (size_t)(next - p) : strlen(p);
		gchar *line = g_strndup(p, lineLen);
		gchar **parts = g_strsplit(line, "\x1f", 2); // label, action
		if (g_strv_length(parts) == 2) {
			GtkWidget *it = gtk_menu_item_new_with_label(parts[0]);
			g_signal_connect_data(it, "activate", G_CALLBACK(hfdMenuItemActivate),
			                      g_strdup(parts[1]), (GClosureNotify)g_free, 0);
			gtk_menu_shell_append(GTK_MENU_SHELL(sub), it);
		}
		g_strfreev(parts);
		g_free(line);
		if (next == NULL) break;
		p = next + 1;
	}

	g_free(menuLabel);
	gtk_box_pack_start(GTK_BOX(box), bar, FALSE, FALSE, 0);
	gtk_box_reorder_child(GTK_BOX(box), bar, 0);
	gtk_widget_show_all(bar);
	hfdMenuBar = bar;
}

// ---- clipboard ---------------------------------------------------------------

static void hfdClipboardSet(const char *text) {
	gtk_clipboard_set_text(gtk_clipboard_get(GDK_SELECTION_CLIPBOARD), text, (gint)strlen(text));
}

static char *hfdClipboardGet(void) {
	return gtk_clipboard_wait_for_text(gtk_clipboard_get(GDK_SELECTION_CLIPBOARD)); // g_free
}

// ---- title -------------------------------------------------------------------

static void hfdSetTitle(const char *title) {
	if (hfdWin != NULL) {
		gtk_window_set_title(GTK_WINDOW(hfdWin), title);
	}
}

// ---- dialogs (modal — chỉ gọi trên main thread) ------------------------------

// Thêm các GtkFileFilter từ chuỗi serialize: "Name\tpat1,pat2\n..."
static void hfdAddFilters(GtkFileChooser *chooser, const char *serialized) {
	if (serialized == NULL || serialized[0] == '\0') return;
	const char *p = serialized;
	while (p != NULL && *p != '\0') {
		const char *nl = strchr(p, '\n');
		size_t lineLen = nl ? (size_t)(nl - p) : strlen(p);
		gchar *line = g_strndup(p, lineLen);
		gchar **parts = g_strsplit(line, "\t", 2);
		if (g_strv_length(parts) == 2) {
			GtkFileFilter *filter = gtk_file_filter_new();
			gtk_file_filter_set_name(filter, parts[0]);
			gchar **pats = g_strsplit(parts[1], ",", -1);
			for (int i = 0; pats[i] != NULL; i++) {
				if (pats[i][0] != '\0') {
					gtk_file_filter_add_pattern(filter, pats[i]);
				}
			}
			g_strfreev(pats);
			gtk_file_chooser_add_filter(chooser, filter);
		}
		g_strfreev(parts);
		g_free(line);
		if (nl == NULL) break;
		p = nl + 1;
	}
}

static char *hfdDialogOpenFile(const char *title, const char *filters, const char *startDir) {
	GtkWidget *dlg = gtk_file_chooser_dialog_new(
		title, GTK_WINDOW(hfdWin), GTK_FILE_CHOOSER_ACTION_OPEN,
		"_Huỷ", GTK_RESPONSE_CANCEL, "_Mở", GTK_RESPONSE_ACCEPT, NULL);

	hfdAddFilters(GTK_FILE_CHOOSER(dlg), filters);
	if (startDir != NULL && startDir[0] != '\0') {
		gtk_file_chooser_set_current_folder(GTK_FILE_CHOOSER(dlg), startDir);
	}

	char *result = NULL;
	if (gtk_dialog_run(GTK_DIALOG(dlg)) == GTK_RESPONSE_ACCEPT) {
		result = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(dlg)); // g_free
	}
	gtk_widget_destroy(dlg);
	return result;
}

static char *hfdDialogSaveFile(const char *title, const char *suggested) {
	GtkWidget *dlg = gtk_file_chooser_dialog_new(
		title, GTK_WINDOW(hfdWin), GTK_FILE_CHOOSER_ACTION_SAVE,
		"_Huỷ", GTK_RESPONSE_CANCEL, "_Lưu", GTK_RESPONSE_ACCEPT, NULL);

	gtk_file_chooser_set_do_overwrite_confirmation(GTK_FILE_CHOOSER(dlg), TRUE);
	if (suggested != NULL && suggested[0] != '\0') {
		gtk_file_chooser_set_current_name(GTK_FILE_CHOOSER(dlg), suggested);
	}

	char *result = NULL;
	if (gtk_dialog_run(GTK_DIALOG(dlg)) == GTK_RESPONSE_ACCEPT) {
		result = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(dlg));
	}
	gtk_widget_destroy(dlg);
	return result;
}

// level: 0 info 1 warning 2 error 3 question(yes/no)
static int hfdDialogMessage(const char *title, const char *message, int level) {
	GtkMessageType t    = GTK_MESSAGE_INFO;
	GtkButtonsType btns = GTK_BUTTONS_OK;
	switch (level) {
		case 1: t = GTK_MESSAGE_WARNING; break;
		case 2: t = GTK_MESSAGE_ERROR; break;
		case 3: t = GTK_MESSAGE_QUESTION; btns = GTK_BUTTONS_YES_NO; break;
	}
	GtkWidget *dlg = gtk_message_dialog_new(GTK_WINDOW(hfdWin), GTK_DIALOG_MODAL,
	                                        t, btns, "%s", message);
	gtk_window_set_title(GTK_WINDOW(dlg), title);
	int resp = gtk_dialog_run(GTK_DIALOG(dlg));
	gtk_widget_destroy(dlg);
	return (resp == GTK_RESPONSE_OK || resp == GTK_RESPONSE_ACCEPT || resp == GTK_RESPONSE_YES) ? 1 : 0;
}
*/
import "C"

import (
	_ "embed"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

//go:embed runtime/hfd.js
var runtimeJS []byte

var assetMIME = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".txt":   "text/plain; charset=utf-8",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".wasm":  "application/wasm",
	".xml":   "application/xml",
}

func mimeFor(name string) string {
	if m, ok := assetMIME[strings.ToLower(path.Ext(name))]; ok {
		return m
	}
	return "application/octet-stream"
}

// scriptMsg là một IPC call nhận từ JS (đã ref reply + JSCContext).
type scriptMsg struct {
	payload string
	pageURI string
	jscCtx  unsafe.Pointer
	reply   unsafe.Pointer
}

type engine struct {
	opts           Options
	view           unsafe.Pointer
	token          string           // per-launch capability token (layer 3)
	allowed        []*regexp.Regexp // AllowOrigins — URI page phải khớp 1 pattern
	schemeRequests chan unsafe.Pointer
	messages       chan scriptMsg
	uiTasks        chan func() // chạy trên GTK main thread thông qua pump
	done           chan struct{}
	closeOnce      sync.Once
}

var _ = fs.FS(nil) // giữ import fs (unix backend dùng trong Options.Assets)

// gtkEngine được assign trong run() trước khi GTK loop chạy (happens-before
// với mọi callback vì callback chỉ phát sinh sau gtk_main đã chạy).
var gtkEngine *engine

// globRegexp chuyển pattern có '*' (bất kỳ chuỗi nào, kể cả '/') → regexp.
func globRegexp(pattern string) *regexp.Regexp {
	parts := strings.Split(pattern, "*")
	var b strings.Builder
	b.WriteString("^")
	for i, p := range parts {
		b.WriteString(regexp.QuoteMeta(p))
		if i < len(parts)-1 {
			b.WriteString(".*")
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		// pattern rất khócompile fail (QuoteMeta an toàn); nếu fail → từ chối.
		return regexp.MustCompile(`^\x00\x00$`)
	}
	return re
}

// newToken sinh token per-launch (crypto/rand 32 byte hex).
func newToken() string {
	buf := make([]byte, 32)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func (e *engine) uriAllowed(uri string) bool {	if len(e.allowed) == 0 {
		return false
	}
	for _, re := range e.allowed {
		if re.MatchString(uri) {
			return true
		}
	}
	return false
}

func newEngine(opts Options) backend {
	return &engine{
		opts:           opts,
		schemeRequests: make(chan unsafe.Pointer, 32),
		messages:       make(chan scriptMsg, 64),
		uiTasks:        make(chan func(), 256),
		done:           make(chan struct{}),
	}
}

// run khóa goroutine này vào OS thread main để GTK khởi tạo đúng.
func (e *engine) run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	gtkEngine = e // callbacks đọc đây sau khi gtk_main chạy

	titleC := C.CString(e.opts.Title)
	defer C.free(unsafe.Pointer(titleC))
	var devURLC *C.char
	if e.opts.DevURL != "" {
		devURLC = C.CString(e.opts.DevURL)
		defer C.free(unsafe.Pointer(devURLC))
	}
	menuC := C.CString(menuSerialized(e.opts))
	defer C.free(unsafe.Pointer(menuC))

	// AllowOrigins: nil → mặc định hfd://app/* (+ devURL/* nếu dev).
	// compile pattern glob '*' thành regexp.
	origins := e.opts.AllowOrigins
	if origins == nil {
		origins = []string{"hfd://app/*"}
		if e.opts.DevURL != "" {
			origins = append(origins, e.opts.DevURL+"/*")
		}
	}
	e.allowed = make([]*regexp.Regexp, 0, len(origins))
	for _, p := range origins {
		e.allowed = append(e.allowed, globRegexp(p))
	}
	allowListC := C.CString(strings.Join(origins, "\n"))
	defer C.free(unsafe.Pointer(allowListC))

	// Per-launch capability token: chỉ runtime của chúng ta biết token
	// (inject qua user script). Mọi IPC phải mang token đúng mới được dùng.
	e.token = newToken()
	runtimeWithCfg := fmt.Sprintf("window.__HFD_CONFIG = { token: %q, csp: %q };", e.token, e.opts.CSP) +
		"\n" + string(runtimeJS)
	runtimeC := C.CString(runtimeWithCfg)
	defer C.free(unsafe.Pointer(runtimeC))

	devtools := 0
	if e.opts.Debug {
		devtools = 1
	}

	e.view = unsafe.Pointer(C.hfdBootstrap(C.int(e.opts.Width), C.int(e.opts.Height), titleC, C.int(devtools), runtimeC, devURLC, menuC, allowListC))

	go e.schemeLoop()
	go e.messageLoop()

	C.gtk_main()
	return nil
}

func (e *engine) isClosed() bool {
	select {
	case <-e.done:
		return true
	default:
		return false
	}
}

func (e *engine) markClosed() {
	e.closeOnce.Do(func() { close(e.done) })
}

// ---- C → Go callbacks -------------------------------------------------------

//export hfdGoOnScriptMessage
func hfdGoOnScriptMessage(payload *C.char, uri *C.char, jscCtx unsafe.Pointer, reply unsafe.Pointer) {
	pageURI := ""
	if uri != nil {
		pageURI = C.GoString(uri)
	}
	gtkEngine.messages <- scriptMsg{
		payload: C.GoString(payload),
		pageURI: pageURI,
		jscCtx:  jscCtx,
		reply:   reply,
	}
	C.g_free(C.gpointer(unsafe.Pointer(payload))) // GoString đã copy sang Go heap
	C.g_free(C.gpointer(unsafe.Pointer(uri)))      // g_strdup — Go đã copy
}

//export hfdGoOnSchemeRequest
func hfdGoOnSchemeRequest(request unsafe.Pointer) {
	gtkEngine.schemeRequests <- request
}

//export hfdGoOnPump
func hfdGoOnPump() {
	gtkEngine.drainUI()
}

//export hfdGoOnWindowClosed
func hfdGoOnWindowClosed() {
	gtkEngine.markClosed()
}

//export hfdGoOnMenu
func hfdGoOnMenu(action *C.char) {
	a := C.GoString(action)
	if curBackend == nil {
		return
	}
	// emit từ UI thread — chỉ marshal + push task (không block)
	go func() { _ = curBackend.emit("menu", map[string]any{"action": a}) }()
}

// menuSerialized.encode Options.Menu về format C đọc: "Label\nlbl\x1faction\n..."
func menuSerialized(opts Options) string {
	if len(opts.Menu) == 0 {
		return ""
	}
	label := opts.MenuLabel
	if label == "" {
		label = "Menu"
	}
	var b strings.Builder
	b.WriteString(label)
	b.WriteByte('\n')
	for _, it := range opts.Menu {
		b.WriteString(sanitizeLabel(it.Label, "?"))
		b.WriteByte(0x1f)
		b.WriteString(sanitizeLabel(it.Action, "unknown"))
		b.WriteByte('\n')
	}
	return b.String()
}

// ---- dispatch handler Go -----------------------------------------------------

func (e *engine) messageLoop() {
	for m := range e.messages {
		if e.isClosed() {
			C.hfdReleaseScriptReply(m.reply, m.jscCtx) // thu hồi im lặng
			continue
		}

		// Security tầng 2: page URI phải khớp AllowOrigins dù bridge nhờ lẫn
		// qua frame nào. (Tầng 1: user script chỉ inject trên origin được phép.)
		if !e.uriAllowed(m.pageURI) {
			if e.opts.Debug {
				fmt.Fprintf(os.Stderr, "hfd [debug]: CHẶN IPC từ origin %q\n", m.pageURI)
			}
			e.replyErr(m, fmt.Sprintf("hfd: origin %q không được phép gọi IPC", m.pageURI))
			continue
		}

		var req ipcRequest
		if err := json.Unmarshal([]byte(m.payload), &req); err != nil {
			e.replyErr(m, fmt.Sprintf("hfd: payload IPC sai định dạng: %v", err))
			continue
		}
		// Tầng 3: capability token — mỗi launch một token, chỉ runtime
		// inject của chúng ta biết. Compare constant-time.
		if subtle.ConstantTimeCompare([]byte(req.Token), []byte(e.token)) != 1 {
			if e.opts.Debug {
				fmt.Fprintf(os.Stderr, "hfd [debug]: TỪ CHỐI IPC (token sai) handler %q\n", req.Name)
			}
			e.replyErr(m, "hfd: token không hợp lệ — bridge chưa thể được xác thực")
			continue
		}
		if req.Name == "" {
			e.replyErr(m, "hfd: thiếu tên handler")
			continue
		}
		h, ok := lookupHandler(req.Name)
		if !ok {
			e.replyErr(m, fmt.Sprintf("hfd: handler %q chưa Bind", req.Name))
			continue
		}
		if e.opts.Debug {
			fmt.Fprintf(os.Stderr, "hfd [debug]: gọi handler %s\n", req.Name)
		}

		// Handler chạy ngoài main thread để không chặn UI.
		go func(m scriptMsg, h Handler, args string) {
			defer func() {
				if r := recover(); r != nil {
					e.replyErr(m, fmt.Sprintf("hfd: handler panic %v", r))
				}
			}()
			result, err := h(args)
			if err != nil {
				e.replyErr(m, err.Error())
				return
			}
			data, jerr := json.Marshal(result)
			if jerr != nil {
				e.replyErr(m, fmt.Sprintf("hfd: marshal kết quả: %v", jerr))
				return
			}
			e.reply(m, data)
		}(m, h, argsOrEmpty(req.Args))
	}
}

func argsOrEmpty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "null"
	}
	return string(raw)
}

// reply — cần chạy trên GTK main thread → đẩy qua uiTasks.
func (e *engine) reply(m scriptMsg, data []byte) {
	if e.isClosed() {
		C.hfdReleaseScriptReply(m.reply, m.jscCtx)
		return
	}
	e.uiTasks <- func() {
		jsonC := C.CString(string(data))
		C.hfdReplyJSON(m.reply, m.jscCtx, jsonC)
		C.free(unsafe.Pointer(jsonC))
	}
}

func (e *engine) replyErr(m scriptMsg, msg string) {
	if e.isClosed() {
		C.hfdReleaseScriptReply(m.reply, m.jscCtx)
		return
	}
	e.uiTasks <- func() {
		msgC := C.CString(msg)
		C.hfdReplyError(m.reply, m.jscCtx, msgC)
		C.free(unsafe.Pointer(msgC))
	}
}

// ---- Go → JS: emit ----------------------------------------------------------

func (e *engine) emit(name string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("hfd: marshal event %q: %w", name, err)
	}
	if e.isClosed() {
		return nil
	}
	script := fmt.Sprintf("window.hfd._emit(%q, %s)", name, string(data))
	e.uiTasks <- func() {
		scriptC := C.CString(script)
		C.hfdEvaluate(e.view, scriptC, C.gssize(len(script)))
		C.free(unsafe.Pointer(scriptC))
	}
	return nil
}

// ---- scheme serving: hfd://app/<path> ---------------------------------------

func (e *engine) schemeLoop() {
	for req := range e.schemeRequests {
		if e.isClosed() {
			continue // request sẽ để WebKit tự dọn khi webview hủy
		}
		e.serveScheme(req)
	}
}

func (e *engine) serveScheme(req unsafe.Pointer) {
	uriC := C.hfdRequestGetURI(req) // internal pointer — không free
	uri := C.GoString(uriC)

	u, err := url.Parse(uri)
	if err != nil {
		e.finishText(req, "hfd:// 400 bad uri")
		return
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if strings.Contains(name, "..") {
		e.finishText(req, "hfd:// 403")
		return
	}

	if name == "__runtime/hfd.js" {
		e.finish(req, runtimeJS, "text/javascript; charset=utf-8")
		return
	}

	var data []byte
	if e.opts.Assets != nil {
		data, err = readFile(e.opts.Assets, name)
		if err != nil {
			e.finishText(req, fmt.Sprintf("hfd:// 404: %s", name))
			return
		}
	} else if name == "index.html" {
		data = []byte(defaultIndexHTML)
	} else {
		e.finishText(req, fmt.Sprintf("hfd:// 404: Assets chưa cấu hình, không tìm thấy %s", name))
		return
	}

	e.finish(req, data, mimeFor(name))
}

func (e *engine) finish(req unsafe.Pointer, data []byte, mime string) {
	e.uiTasks <- func() {
		dataC := C.CBytes(data)
		mimeC := C.CString(mime)
		C.hfdFinishRequest(req, dataC, C.gssize(len(data)), mimeC)
		C.free(unsafe.Pointer(dataC))
		C.free(unsafe.Pointer(mimeC))
	}
}

func (e *engine) finishText(req unsafe.Pointer, text string) {
	e.finish(req, []byte(text), "text/plain; charset=utf-8")
}

func readFile(fsys fs.FS, name string) ([]byte, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

const defaultIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>HFD</title>
<style>body{font-family:sans-serif;background:#16181d;color:#e8e9ec;display:grid;place-items:center;height:100vh;margin:0}code{background:#24272e;padding:2px 6px;border-radius:4px}</style>
</head><body><div><h1>HFD chạy rồi</h1><p>Đây là trang index mặc định — hãy cấu hình <code>Options.Assets</code> để nạp app của bạn.</p></div></body></html>`

// ---- pump: đợi + xả task trên main thread -----------------------------------

func (e *engine) drainUI() {
	for {
		select {
		case f := <-e.uiTasks:
			if e.isClosed() {
				return
			}
			f()
		default:
			return
		}
	}
}

// ---- native API: clipboard / dialog / title / close --------------------------
// Methods đẩy task lên main thread (GTK thao tác UI duy nhất ở đó) và chờ
// kết quả. App đóng giữa chừng → trả lỗi "app đã đóng", không deadlock.

func (e *engine) onMain[T any](run func() T) (T, error) {
	var zero T
	if e.isClosed() {
		return zero, fmt.Errorf("hfd: app đã đóng")
	}
	ch := make(chan T, 1)
	e.uiTasks <- func() { ch <- run() } // uiTasks buffer 256 — non-blocking
	select {
	case v := <-ch:
		return v, nil
	case <-e.done:
		return zero, fmt.Errorf("hfd: app đã đóng khi chờ UI")
	}
}

func (e *engine) setClipboard(text string) error {
	_, err := e.onMain(func() struct{} {
		cs := C.CString(text)
		C.hfdClipboardSet(cs)
		C.free(unsafe.Pointer(cs))
		return struct{}{}
	})
	return err
}

func (e *engine) getClipboard() (string, error) {
	str, err := e.onMain(func() *C.char {
		return C.hfdClipboardGet()
	})
	if err != nil {
		return "", err
	}
	if str == nil {
		return "", nil
	}
	defer C.g_free(C.gpointer(unsafe.Pointer(str)))
	return C.GoString(str), nil
}

func (e *engine) openFileDialog(title, serializedFilters, startDir string) (string, bool, error) {
	path, err := e.onMain(func() *C.char {
		t := C.CString(title)
		defer C.free(unsafe.Pointer(t))
		f := C.CString(serializedFilters)
		defer C.free(unsafe.Pointer(f))
		d := C.CString(startDir)
		defer C.free(unsafe.Pointer(d))
		return C.hfdDialogOpenFile(t, f, d)
	})
	if err != nil {
		return "", false, err
	}
	if path == nil {
		return "", false, nil // user huỷ
	}
	defer C.g_free(C.gpointer(unsafe.Pointer(path)))
	return C.GoString(path), true, nil
}

func (e *engine) saveFileDialog(title, suggestedName string) (string, bool, error) {
	path, err := e.onMain(func() *C.char {
		t := C.CString(title)
		defer C.free(unsafe.Pointer(t))
		n := C.CString(suggestedName)
		defer C.free(unsafe.Pointer(n))
		return C.hfdDialogSaveFile(t, n)
	})
	if err != nil {
		return "", false, err
	}
	if path == nil {
		return "", false, nil
	}
	defer C.g_free(C.gpointer(unsafe.Pointer(path)))
	return C.GoString(path), true, nil
}

func (e *engine) showMessage(title, message string, level int) (bool, error) {
	ok, err := e.onMain(func() C.int {
		t := C.CString(title)
		defer C.free(unsafe.Pointer(t))
		m := C.CString(message)
		defer C.free(unsafe.Pointer(m))
		return C.hfdDialogMessage(t, m, C.int(level))
	})
	return ok == 1, err
}

func (e *engine) setTitle(title string) error {
	_, err := e.onMain(func() struct{} {
		t := C.CString(title)
		C.hfdSetTitle(t)
		C.free(unsafe.Pointer(t))
		return struct{}{}
	})
	return err
}

func (e *engine) closeWindow() error {
	_, err := e.onMain(func() struct{} {
		C.hfdCloseWindow()
		return struct{}{}
	})
	return err
}
