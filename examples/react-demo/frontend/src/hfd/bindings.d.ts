// TỰ SINH bởi `hfd bindings` — KHÔNG sửa tay.
// Source of truth: các hàm hfd.BindTyped trong Go.

export interface GreetArgs {
	name: string;
}
export interface GreetResult {
	message: string;
}
export interface SysInfoArgs {
	// struct rỗng
}
export interface SysInfoResult {
	hostname: string;
	os: string;
	arch: string;
	cpus: number;
	goVersion: string;
}

export interface HFDHandlers {
	Greet: [GreetArgs, GreetResult];
	SysInfo: [SysInfoArgs, SysInfoResult];
}

export declare function call<K extends keyof HFDHandlers>(
	name: K,
	args: HFDHandlers[K][0],
): Promise<HFDHandlers[K][1]>;
export declare function on(name: string, cb: (data: unknown) => void): () => void;
export declare function off(name: string, cb: (data: unknown) => void): void;
