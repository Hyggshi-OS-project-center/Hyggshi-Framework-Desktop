// @hfd/react — hooks for HFD: call Go handlers type-safe + listen to Go→JS events.
//
// Requires: the Go core has injected the JS runtime (window.hfd) and the
// bindings codegen for strict types via the HFDHandlers map.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

/** Call a Go handler (wraps window.hfd.call). */
export function call(name, args) {
	return window.hfd.call(name, args);
}

/** Subscribe to a Go → JS event; returns an unsubscribe function. */
export function on(name, cb) {
	return window.hfd.on(name, cb);
}

/**
 * useHfdEvent — listen to a Go → JS event.
 * Changing the callback does not re-subscribe (kept in an internal ref).
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
 * useHfdCall — calls a Go handler on mount and refetches when args change
 * (matched via the JSON key). Returns { data, error, loading, refetch }.
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
