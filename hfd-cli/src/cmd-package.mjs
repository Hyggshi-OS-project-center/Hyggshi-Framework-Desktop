// `hfd package [dir]` — build the app, then assemble an installable .deb
// (usr/local/bin + a .desktop launcher). Uses dpkg-deb; no other deps.
import { spawnSync } from "node:child_process";
import { mkdirSync, rmSync, writeFileSync, cpSync, existsSync, readFileSync, readdirSync } from "node:fs";
import { join, resolve } from "node:path";
import { runBuild } from "./cmd-build.mjs";
import { loadConfig, goEnv } from "./config.mjs";

function sh(cmd, args, opts = {}) {
	const r = spawnSync(cmd, args, { encoding: "utf8", env: goEnv(), ...opts });
	if (r.status !== 0) {
		throw new Error(`${cmd} failed:\n${(r.stderr || r.stdout || "").trim()}`);
	}
	return r;
}

function desktopEntry(name, binaryName, comment) {
	return `[Desktop Entry]
Type=Application
Version=1.0
Name=${name}
Comment=${comment || `${name} — built with HFD`}
Exec=/usr/local/bin/${binaryName}
Icon=applications-utilities
Terminal=false
Categories=Development;Utility;
`;
}

// Find the built binary under app dir build/bin/ (goBuild decides its name).
function findBinary(cfg) {
	const binDir = join(cfg.appDir, "build", "bin");
	if (!existsSync(binDir)) return null;
	for (const f of readdirSync(binDir)) {
		if (!f.endsWith(".test") && !f.endsWith(".log")) return join(binDir, f);
	}
	return null;
}

export function runPackage(dir) {
	const cfg = loadConfig(dir);
	let name = "hfd-app";
	let comment = "";
	try {
		const j = JSON.parse(readFileSync(join(cfg.appDir, "hfd.json"), "utf8"));
		name = j.package?.name || j.app?.name || name;
		comment = j.package?.description || "";
	} catch { /* defaults fine */ }
	if (name === "hfd-app") {
		// fall back to the Go module name (hfd-code, demo, …)
		const goMod = join(cfg.appDir, "go.mod");
		if (existsSync(goMod)) {
			const m = readFileSync(goMod, "utf8").match(/^module\s+(\S+)/m);
			if (m && m[1] !== "hfd-app") name = m[1];
		}
	}

	console.log(`▸ package app ${cfg.appDir}`);
	runBuild(dir); // bindings + frontend + go build

	const binary = findBinary(cfg);
	if (!binary) throw new Error("no binary after build (expected build/bin/<name>)");
	const binaryName = name;

	const pkgRoot = resolve(cfg.appDir, "build", "deb");
	rmSync(pkgRoot, { recursive: true, force: true });
	mkdirSync(join(pkgRoot, "DEBIAN"), { recursive: true });
	mkdirSync(join(pkgRoot, "usr", "local", "bin"), { recursive: true });
	mkdirSync(join(pkgRoot, "usr", "share", "applications"), { recursive: true });

	cpSync(binary, join(pkgRoot, "usr", "local", "bin", binaryName));
	writeFileSync(
		join(pkgRoot, "usr", "share", "applications", `${binaryName}.desktop`),
		desktopEntry(name, binaryName, comment),
	);

	// control: architecture from dpkg
	const arch = sh("dpkg", ["--print-architecture"]).stdout.trim();
	writeFileSync(join(pkgRoot, "DEBIAN", "control"),
		`Package: ${binaryName}\n` +
		`Version: 0.1.0\n` +
		`Section: utils\n` +
		`Priority: optional\n` +
		`Architecture: ${arch}\n` +
		`Maintainer: Hyggshi <hyggshidev@gmail.com>\n` +
		`Description: ${comment || `${name} desktop app built with HFD`}\n` +
		` Hyggshi Framework Desktop (Go + GTK/WebKitGTK).\n`);

	const deb = resolve(cfg.appDir, "build", `${binaryName}_0.1.0_${arch}.deb`);
	sh("dpkg-deb", ["--root-owner-group", "--build", pkgRoot, deb]);
	const info = sh("dpkg-deb", ["--info", deb]).stdout;
	console.log(`✓ package built: ${deb}`);
	console.log(info.trim().split("\n").slice(0, 4).join("\n"));
	return deb;
}
