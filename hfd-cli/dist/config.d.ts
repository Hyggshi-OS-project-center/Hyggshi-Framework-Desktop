export interface FrontendConfigJSON {
    dir?: string;
    dev?: string;
    build?: string;
    port?: number | string;
}
export interface HfdJSON {
    app?: {
        dir?: string;
        name?: string;
    };
    frontend?: FrontendConfigJSON;
    bindingsDir?: string;
    goBuild?: string;
    package?: {
        name?: string;
        description?: string;
    };
}
export interface HfdConfig {
    /** project root — the directory passed on the command line, resolved */
    root: string;
    /** directory holding go.mod (defaults to root) */
    appDir: string;
    frontendDir: string;
    /** frontend dev-server command, e.g. "npm run dev" (null = none) */
    frontendDev: string | null;
    /** frontend build command (null = nothing to build) */
    frontendBuild: string | null;
    frontendPort: number;
    /** output dir for the codegen bindings (bindings.js + bindings.d.ts) */
    bindingsDir: string;
    /** go build command (default inferred from the go.mod module name) */
    goBuild: string;
}
/** Env for Go processes — appends user-local Go install paths to PATH. */
export declare function goEnv(): NodeJS.ProcessEnv;
/** Reads the project's hfd.json (optional) and infers the go build command. */
export declare function loadConfig(dir: string): HfdConfig;
