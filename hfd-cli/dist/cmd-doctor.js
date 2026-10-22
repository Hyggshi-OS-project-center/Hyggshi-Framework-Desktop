// cmd-doctor.ts — `hfd doctor [dir]`
//
// Environment check for developing and building HFD apps. Zero deps:
// everything through spawnSync probes.
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { goEnv, loadConfig } from "./config.js";
function probe(cmd, args) {
    const r = spawnSync(cmd, args, { encoding: "utf8", env: goEnv(), shell: false });
    if (r.error || r.status !== 0)
        return null;
    return (r.stdout ?? "").trim().split("\n")[0] ?? null;
}
function check(label, ok, detail, hint) {
    const mark = ok ? "✓" : "✗";
    console.log(`${mark} ${label}${detail ? ` — ${detail}` : ""}`);
    if (!ok && hint)
        console.log(`    hint: ${hint}`);
    return ok;
}
export function runDoctor(dir) {
    const cfg = loadConfig(dir);
    const env = goEnv();
    console.log(`▸ hfd doctor — app ${cfg.appDir}\n`);
    // 1. Go toolchain
    const goVer = probe("go", ["version"]);
    check("Go toolchain", goVer !== null, goVer ?? "not found", "install Go (or set PATH); hfd-cli also probes ~/.local/go/bin");
    // 2. Node (needed by hfd-cli itself / vite dev)
    const nodeVer = probe("node", ["--version"]);
    check("Node.js", nodeVer !== null, nodeVer ?? "not found", "node >= 18 recommended");
    // 3. Linux native stack
    if (process.platform === "linux") {
        const gtk = probe("pkg-config", ["--modversion", "gtk+-3.0"]);
        check("GTK3 dev (gtk+-3.0)", gtk !== null, gtk ?? "missing pkg-config/package", "sudo apt install libgtk-3-dev");
        const webkit = probe("pkg-config", ["--modversion", "webkit2gtk-4.1"]);
        check("WebKitGTK 4.1 dev", webkit !== null, webkit ?? "missing", "sudo apt install libwebkit2gtk-4.1-dev libsoup-3.0-dev");
        const soup = probe("pkg-config", ["--modversion", "libsoup-3.0"]);
        check("libsoup 3.0 dev", soup !== null, soup ?? "missing", "sudo apt install libsoup-3.0-dev");
        // 4. tray host (optional)
        let watcher = false;
        try {
            const busAddr = env.DBUS_SESSION_BUS_ADDRESS ||
                `unix:path=/run/user/${process.getuid?.() ?? 1000}/bus`;
            const r = spawnSync("dbus-send", [
                "--session", "--print-reply",
                "--dest=org.freedesktop.DBus", "/org/freedesktop/DBus",
                "org.freedesktop.DBus.ListNames",
            ], { encoding: "utf8", env: { ...env, DBUS_SESSION_BUS_ADDRESS: busAddr } });
            watcher = (r.stdout ?? "").includes("org.kde.StatusNotifierWatcher");
        }
        catch {
            // absent dbus-send → treat as unknown
        }
        console.log(`${watcher ? "✓" : "-"} tray host (org.kde.StatusNotifierWatcher)` +
            (watcher ? "" : " — not detected; tray still builds (-tags hfd_tray) but GNOME needs the AppIndicator extension"));
    }
    else {
        console.log(`- platform ${process.platform}: Linux native stack not required`);
        console.log("  Windows/macOS backends are Phase 5 — see docs/PHASE5-BACKENDS.md");
    }
    // 5. project config sanity
    console.log("");
    let allGood = true;
    if (!existsSync(join(cfg.appDir, "go.mod"))) {
        allGood = false;
        check("project go.mod", false, "missing", "is this an HFD app dir?");
    }
    else {
        check("project go.mod", true, join(cfg.appDir, "go.mod"));
    }
    if (!existsSync(join(cfg.appDir, "hfd.json"))) {
        console.log("- hfd.json absent — CLI defaults apply (fine for scaffolds)");
    }
    else {
        try {
            JSON.parse(readFileSync(join(cfg.appDir, "hfd.json"), "utf8"));
            check("hfd.json parses", true);
        }
        catch (e) {
            allGood = false;
            check("hfd.json parses", false, e instanceof Error ? e.message : String(e), "fix the JSON syntax");
        }
    }
    if (cfg.frontendDev !== null && !existsSync(join(cfg.frontendDir, "package.json"))) {
        allGood = false;
        check("frontend package.json", false, "frontend.dev is set but package.json is missing");
    }
    console.log(allGood ? "\n✓ environment looks good" : "\n✗ some checks need attention");
    return allGood ? 0 : 1;
}
