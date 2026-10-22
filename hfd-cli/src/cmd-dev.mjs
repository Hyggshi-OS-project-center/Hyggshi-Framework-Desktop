import { spawn } from "node:child_process";
import http from "node:http";
import process from "node:process";
import { loadConfig, goEnv } from "./config.mjs";

// `hfd dev [dir]` — starts the frontend dev server, waits for the port, then
// runs the Go app with HFD_DEV_URL=http://localhost:<port> (the webview loads
// the dev server URL; the JS runtime is still injected by Go).
export function runDev(dir) {
	const cfg = loadConfig(dir);

	const goProc = { ref: null };
	const feProc = { ref: null };

	const cleanup = () => {
		for (const p of [goProc.ref, feProc.ref]) {
			if (p && p.exitCode === null) p.kill("SIGTERM");
		}
	};
	process.on("SIGINT", cleanup);
	process.on("SIGTERM", cleanup);

	const startGo = () => {
		const hasDevServer = !!cfg.frontendDev;
		const env = { ...process.env };
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

	const devURL = `http://localhost:${cfg.frontendPort}`;

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

function waitPort(port, timeoutMs) {
	return new Promise((resolve, reject) => {
		const started = Date.now();
		const probe = () => {
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
