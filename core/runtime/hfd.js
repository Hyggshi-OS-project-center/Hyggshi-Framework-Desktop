/**
 * Hyggshi Framework Desktop — JS runtime (embedded into the webview, no npm)
 *
 * Mechanics:
 * - hfd.call(name, args) → postMessage JSON qua script message handler "hfd"
 *   (register_script_message_handler_with_reply) → WebKit delivers a REAL Promise;
 *   Go completes the handler and replies via WebKitScriptMessageReply.
 * - hfd.on(name, cb) → register listener; Go fires events via evaluate_javascript("_emit").
 */
(() => {
	"use strict";

	if (window.hfd) return; // idempotent — never inject twice

	const cfg = window.__HFD_CONFIG || { token: "", csp: "" };

	// CSP: insert <meta http-equiv=Content-Security-Policy> very early (DOCUMENT_START
	// runs before any page script) so the CSP covers the whole page.
	if (cfg.csp) {
		try {
			const meta = document.createElement("meta");
			meta.httpEquiv = "Content-Security-Policy";
			meta.content = cfg.csp;
			(document.head || document.documentElement).appendChild(meta);
		} catch (e) {
			console.error("[hfd] CSP injection error:", e);
		}
	}

	console.log("[hfd] runtime injected via user script");
	const listeners = new Map();

	window.hfd = {
		/**
		 * Call a Bind'ed Go handler. Returns a Promise (WebKit native reply).
		 * @param {string} name  handler name
		 * @param {any}    args  payload (JSON-serializable)
		 */
		call(name, args) {
			return window.webkit.messageHandlers.hfd.postMessage(
				JSON.stringify({ type: "call", name, args: args === undefined ? null : args, token: cfg.token })
			);
		},

		/**
		 * Register a listener for a Go → JS event.
		 * @param {string}   name
		 * @param {Function} cb
		 * @returns {Function} unsubscribe function
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

		/** Internal: Go fires events here (via evaluate_javascript) */
		_emit(name, data) {
			for (const cb of [...(listeners.get(name) || [])]) {
				try { cb(data); } catch (e) {
					console.error(`hfd: listener "${name}" error`, e);
				}
			}
		},
	};
})();
