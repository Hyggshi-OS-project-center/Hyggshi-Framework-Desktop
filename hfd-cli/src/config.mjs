import { existsSync } from "node:fs";
import { readFileSync } from "node:fs";
import { join, resolve } from "node:path";

// goEnv: env cho process Go — adiciona các vị trí cài go user-local nếu PATH thiếu.
export function goEnv() {
	const env = { ...process.env };
	const extra = [
		`${env.HOME}/.local/go/bin`,
		`${env.HOME}/go/bin`,
		"/usr/local/go/bin",
	].filter((p) => existsSync(p));
	env.PATH = [...extra, env.PATH || ""].join(":");
	return env;
}

// Đọc hfd.json của project (tùy chọn) + đảm bảo go.mod tồn tại.
export function loadConfig(dir) {
	const root = resolve(process.cwd(), dir);
	const cfgPath = join(root, "hfd.json");

	const cfg = {
		root,
		appDir: root,
		frontendDir: join(root, "frontend"),
		frontendDev: null, // lệnh dev server, vd "npm run dev"
		frontendBuild: null, // lệnh build frontend
		frontendPort: 5173,
		bindingsDir: join(root, "frontend", "src", "hfd"),
		goBuild: null, // mặc định suy từ go.mod: go build -o build/bin/<module> .
	};

	try {
		const user = JSON.parse(readFileSync(cfgPath, "utf8"));
		if (user.app?.dir) cfg.appDir = resolve(root, user.app.dir);
		if (user.frontend?.dir) cfg.frontendDir = resolve(root, user.frontend.dir);
		if (user.frontend?.dev) cfg.frontendDev = user.frontend.dev;
		if (user.frontend?.build) cfg.frontendBuild = user.frontend.build;
		if (user.frontend?.port) cfg.frontendPort = Number(user.frontend.port) || 5173;
		if (user.bindingsDir) cfg.bindingsDir = resolve(root, user.bindingsDir);
		if (user.goBuild) cfg.goBuild = user.goBuild;
	} catch {
		// không có hfd.json — dùng defaults
	}

	if (!cfg.goBuild) {
		// suy tên binary từ go.mod module
		let moduleName = "app";
		try {
			const gomod = readFileSync(join(cfg.appDir, "go.mod"), "utf8");
			moduleName = gomod.match(/^module\s+(\S+)/m)?.[1]?.split("/").pop() || "app";
		} catch {
			throw new Error(`không tìm thấy go.mod trong ${cfg.appDir} — có phải project HFD?`);
		}
		cfg.goBuild = `go build -o build/bin/${moduleName} .`;
	}

	return cfg;
}
