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
	tags: string[];
}
export interface StartTickArgs {
	times: number;
}
export interface StartTickResult {
	started: boolean;
}
export interface PickFileArgs {
	title: string;
}
export interface PickFileResult {
	path: string;
	canceled: boolean;
}
export interface SaveTextArgs {
	name: string;
	text: string;
}
export interface SaveTextResult {
	path: string;
	bytes: number;
	canceled: boolean;
}
export interface SetTextArgs {
	text: string;
}
export interface SetTextResult {
	saved: boolean;
}
export interface EchoResult {
	text: string;
}
export interface NotifyArgs {
	title: string;
	body: string;
}
export interface NotifyResult {
	sent: boolean;
	err: string;
}

export interface HFDHandlers {
	Greet: [GreetArgs, GreetResult];
	SysInfo: [SysInfoArgs, SysInfoResult];
	StartTick: [StartTickArgs, StartTickResult];
	PickFile: [PickFileArgs, PickFileResult];
	SaveText: [SaveTextArgs, SaveTextResult];
	ClipboardSet: [SetTextArgs, SetTextResult];
	ClipboardGet: [SetTextArgs, EchoResult];
	SendNotification: [NotifyArgs, NotifyResult];
	AskConfirm: [SetTextArgs, EchoResult];
}

export declare function call<K extends keyof HFDHandlers>(
	name: K,
	args: HFDHandlers[K][0],
): Promise<HFDHandlers[K][1]>;
export declare function on(name: string, cb: (data: unknown) => void): () => void;
export declare function off(name: string, cb: (data: unknown) => void): void;
