// @hfd/react — TypeScript types.
//
// Phiên bản generic mềm: app có thể narrow qua HFDHandlers từ bindings codegen
// (hfd bindings) bằng wrapper type riêng, ví dụ:
//
//   type Call = <K extends keyof HFDHandlers>(n: K, a: HFDHandlers[K][0])
//     => Promise<HFDHandlers[K][1]>;

export declare function call(name: string, args?: unknown): Promise<unknown>;

export declare function on(name: string, cb: (data: unknown) => void): () => void;

export declare function useHfdEvent(name: string, cb: (data: unknown) => void): void;

export declare function useHfdCall<In = any, Out = unknown>(
	name: string,
	args?: In
): UseHfdCallResult<Out>;

export interface UseHfdCallResult<Out> {
	data: Out | undefined;
	error: Error | undefined;
	loading: boolean;
	refetch: () => Promise<void>;
}
