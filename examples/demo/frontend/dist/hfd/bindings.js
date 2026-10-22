// TỰ SINH bởi `hfd bindings` — glue quanh window.hfd (runtime Go inject).
if (typeof window !== "undefined") {
	window.hfdAPI = {
		call(name, args) { return window.hfd.call(name, args); },
		on(name, cb) { return window.hfd.on(name, cb); },
		off(name, cb) { window.hfd.off(name, cb); },
	};
}
export function call(name, args) {
	return window.hfd.call(name, args);
}
export function on(name, cb) {
	return window.hfd.on(name, cb);
}
export function off(name, cb) {
	window.hfd.off(name, cb);
}
