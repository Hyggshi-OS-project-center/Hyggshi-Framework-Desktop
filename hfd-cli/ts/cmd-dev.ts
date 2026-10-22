// cmd-dev.ts — `hfd dev [dir]`
//
// Starts the frontend dev server, waits for the port, then runs the Go app
// with HFD_DEV_URL=http://localhost:<port> (the webview loads the dev server
// URL; the JS runtime is still injected by Go).
import { spawn, type ChildProcess } from "node:child_process";
import http from "node:http";
import process from "node:process";
import { loadConfig, goEnv } from "./config.js";

export async function runDev(dir: string): Promise<void> {
	const cfg = loadConfig(dir);
	const devURL = `http://localhost:${cfg.frontendPort}`;

	const goProc: { ref: ChildProcess | null } = { ref: null };
	const feProc: { ref: ChildProcess | null } = { ref: null };

	const cleanup = (): void => {
		for (const p of [goProc.ref, feProc.ref]) {
			if (p && p.exitCode === null) p.kill("SIGTERM");
		}
	};
	process.on("SIGINT", cleanup);
	process.on("SIGTERM", cleanup);

	const startGo = (): void => {
		const hasDevServer = cfg.frontendDev !== null;
		const env: NodeJS.ProcessEnv = { ...process.env };
		if (hasDevServer) env.HFD_DEV_URL = devURL;
		console.log(`▸ running Go app${hasDevServer ? ` with dev server ${devURL}` : ""}`);
		const p = spawn("go", ["run", "."], {
			cwd: cfg.appDir,
			stdio: "inherit",
			env: { ...goEnv(), ...env },
		});
		goProc.ref = p;
		p.on("exit", () => cleanup());
	};

	if (cfg.frontendDev) {
		console.log(`▸ dev server: ${cfg.frontendDev} (cwd ${cfg.frontendDir})`);
		const p = spawn(cfg.frontendDev, {
			cwd: cfg.frontendDir,
			stdio: ["ignore", "inherit", "inherit"],
			shell: true,
		});
		feProc.ref = p;
		p.on("exit", () => cleanup());

		waitPort(cfg.frontendPort, 20000)
			.then(startGo)
			.catch(() => {
				console.error("hfd: dev server did not come up in time — running the Go app anyway");
				startGo();
			});
	} else {
		startGo();
	}
}

function waitPort(port: number, timeoutMs: number): Promise<void> {
	return new Promise((resolve, reject) => {
		const started = Date.now();
		const probe = (): void => {
			const req = http.get(`http://localhost:${port}`, { timeout: 1000 }, (res) => {
				res.resume();
				resolve();
			});
			req.on("error", () => {
				if (Date.now() - started > timeoutMs) {
					reject(new Error("timeout"));
					return;
				}
				setTimeout(probe, 300);
			});
		};
		probe();
	});
}
