// @hfd/react — TypeScript types.
//
// Soft generic version: apps can narrow it via HFDHandlers from the bindings
// codegen (hfd bindings) with their own wrapper types, e.g.:
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
