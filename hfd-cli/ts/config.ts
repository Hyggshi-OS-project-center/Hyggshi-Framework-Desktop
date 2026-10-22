// config.ts — reads hfd.json and provides the Go process environment.
//
// Zero runtime deps: node built-ins only. The TypeScript dev toolchain
// (typescript, @types/node) is needed only to REBUILD the CLI, never to run it.
import { existsSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";

// ---- types ------------------------------------------------------------------

export interface FrontendConfigJSON {
	dir?: string;
	dev?: string;
	build?: string;
	port?: number | string;
}

export interface HfdJSON {
	app?: { dir?: string; name?: string };
	frontend?: FrontendConfigJSON;
	bindingsDir?: string;
	goBuild?: string;
	package?: { name?: string; description?: string };
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

// ---- goEnv ------------------------------------------------------------------

/** Env for Go processes — appends user-local Go install paths to PATH. */
export function goEnv(): NodeJS.ProcessEnv {
	const env: NodeJS.ProcessEnv = { ...process.env };
	const home = env.HOME ?? "";
	const extra = [
		`${home}/.local/go/bin`,
		`${home}/go/bin`,
		"/usr/local/go/bin",
	].filter((p) => p !== "" && existsSync(p));
	env.PATH = [...extra, env.PATH ?? ""].join(":");
	return env;
}

// ---- loadConfig -------------------------------------------------------------

/** Reads the project's hfd.json (optional) and infers the go build command. */
export function loadConfig(dir: string): HfdConfig {
	const root = resolve(process.cwd(), dir);
	const cfgPath = join(root, "hfd.json");

	const cfg: HfdConfig = {
		root,
		appDir: root,
		frontendDir: join(root, "frontend"),
		frontendDev: null,
		frontendBuild: null,
		frontendPort: 5173,
		bindingsDir: join(root, "frontend", "src", "hfd"),
		goBuild: "",
	};

	try {
		const user = JSON.parse(readFileSync(cfgPath, "utf8")) as HfdJSON;
		if (user.app?.dir) cfg.appDir = resolve(root, user.app.dir);
		if (user.frontend?.dir) cfg.frontendDir = resolve(root, user.frontend.dir);
		if (user.frontend?.dev) cfg.frontendDev = user.frontend.dev;
		if (user.frontend?.build) cfg.frontendBuild = user.frontend.build;
		if (user.frontend?.port) cfg.frontendPort = Number(user.frontend.port) || 5173;
		if (user.bindingsDir) cfg.bindingsDir = resolve(root, user.bindingsDir);
		if (user.goBuild) cfg.goBuild = user.goBuild;
	} catch {
		// no (or invalid) hfd.json — use the defaults
	}

	if (!cfg.goBuild) {
		// infer the binary name from the go.mod module
		let moduleName = "app";
		try {
			const gomod = readFileSync(join(cfg.appDir, "go.mod"), "utf8");
			moduleName = gomod.match(/^module\s+(\S+)/m)?.[1]?.split("/").pop() ?? "app";
		} catch {
			throw new Error(`go.mod not found in ${cfg.appDir} — is this an HFD project?`);
		}
		cfg.goBuild = `go build -o build/bin/${moduleName} .`;
	}

	return cfg;
}
