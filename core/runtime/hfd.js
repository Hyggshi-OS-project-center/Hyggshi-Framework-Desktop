/**
 * Hyggshi Framework Desktop — JS runtime (nhúng vào webview, không cần npm)
 *
 * Cơ chế:
 * - hfd.call(name, args) → postMessage JSON qua script message handler "hfd"
 *   (register_script_message_handler_with_reply) → WebKit trả Promise THẬT,
 *   Go hoàn thành handler và gửi reply qua WebKitScriptMessageReply.
 * - hfd.on(name, cb) → đăng ký listener; Go bắn event qua evaluate_javascript("_emit").
 */
(() => {
	"use strict";

	if (window.hfd) return; // idempotent — không inject 2 lần

	const cfg = window.__HFD_CONFIG || { token: "", csp: "" };

	// CSP: chèn <meta http-equiv=Content-Security-Policy> rất sớm (DOCUMENT_START
	// chạy trước Script(docs)) nên CSP áp toàn trang, kể cả inline script sau.
	if (cfg.csp) {
		try {
			const meta = document.createElement("meta");
			meta.httpEquiv = "Content-Security-Policy";
			meta.content = cfg.csp;
			(document.head || document.documentElement).appendChild(meta);
		} catch (e) {
			console.error("[hfd] CSP injection lỗi:", e);
		}
	}

	console.log("[hfd] runtime injected qua user script");
	const listeners = new Map();

	window.hfd = {
		/**
		 * Gọi một handler Go đã Bind. Trả về Promise (native reply của WebKit).
		 * @param {string} name  tên handler
		 * @param {any}    args  payload (JSON-serializable)
		 */
		call(name, args) {
			return window.webkit.messageHandlers.hfd.postMessage(
				JSON.stringify({ type: "call", name, args: args === undefined ? null : args, token: cfg.token })
			);
		},

		/**
		 * Đăng ký listener cho event Go → JS.
		 * @param {string}   name
		 * @param {Function} cb
		 * @returns {Function} hàm unsubscribe
		 */
		on(name, cb) {
			if (!listeners.has(name)) listeners.set(name, []);
			listeners.get(name).push(cb);
			return () => this.off(name, cb);
		},

		off(name, cb) {
			const list = listeners.get(name) || [];
			const i = list.indexOf(cb);
			if (i >= 0) list.splice(i, 1);
		},

		/** Nội bộ: Go bắn event vào đây (qua evaluate_javascript) */
		_emit(name, data) {
			for (const cb of [...(listeners.get(name) || [])]) {
				try { cb(data); } catch (e) {
					console.error(`hfd: listener "${name}" lỗi`, e);
				}
			}
		},
	};
})();
