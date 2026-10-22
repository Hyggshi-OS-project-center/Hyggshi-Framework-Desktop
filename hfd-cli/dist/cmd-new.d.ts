export type Template = "react" | "vue";
export interface NewOptions {
    template?: string;
    corePath?: string;
}
export declare function runNew(name: string, opts?: NewOptions): Promise<void>;
