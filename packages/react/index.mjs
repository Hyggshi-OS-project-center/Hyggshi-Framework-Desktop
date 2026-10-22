// @hfd/react — hooks cho HFD: gọi handler Go type-safe + lắng nghe event Go→JS.
//
// Yêu cầu: runtime JS đã được Go core inject (window.hfd) và bindings codegen
// cho types chặt qua HFDHandlers map.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

/** Gọi handler Go (wrap window.hfd.call). */
export function call(name, args) {
	return window.hfd.call(name, args);
}

/** Đăng ký event Go → JS; trả về hàm unsubscribe. */
export function on(name, cb) {
	return window.hfd.on(name, cb);
}

/**
 * useHfdEvent — lắng nghe event Go → JS.
 * Callback thay đổi không đăng ký lại (giữ ref nội bộ).
 *
 * @example
 * useHfdEvent("tick", (t) => setCount(t.count));
 */
export function useHfdEvent(name, cb) {
	const cbRef = useRef(cb);
	useEffect(() => {
		cbRef.current = cb;
	});

	useEffect(() => {
		return window.hfd.on(name, (data) => cbRef.current(data));
	}, [name]);
}

/**
 * useHfdCall — gọi handler Go khi mount, tự gọi lại khi args đổi
 * (so khớp bằng JSON key). Trả về { data, error, loading, refetch }.
 *
 * @example
 * const { data, loading } = useHfdCall("SysInfo", {});
 */
export function useHfdCall(name, args = null) {
	const argsKey = useMemo(() => JSON.stringify(args ?? null), [args]);
	const [state, setState] = useState({
		data: undefined,
		error: undefined,
		loading: true,
	});

	const run = useCallback(async () => {
		try {
			const result = await window.hfd.call(name, JSON.parse(argsKey));
			setState({ data: result, error: undefined, loading: false });
		} catch (err) {
			setState({ data: undefined, error: err, loading: false });
		}
	}, [name, argsKey]);

	useEffect(() => {
		run();
	}, [run]);

	return { ...state, refetch: run };
}
