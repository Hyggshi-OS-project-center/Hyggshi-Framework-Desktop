// cmd-build.ts — `hfd build [dir]`
//
// Builds the frontend (if a build command is configured), then runs the Go
// build. Output lands wherever goBuild points (default build/bin/<module>).
import { spawnSync } from "node:child_process";
import { loadConfig, goEnv } from "./config.js";
export async function runBuild(dir) {
    const cfg = loadConfig(dir);
    if (cfg.frontendBuild) {
        console.log(`▸ build frontend: ${cfg.frontendBuild}`);
        const r = spawnSync(cfg.frontendBuild, {
            cwd: cfg.frontendDir,
            stdio: "inherit",
            shell: true,
        });
        if (r.status !== 0)
            throw new Error("frontend build failed");
    }
    console.log(`▸ build app Go: ${cfg.goBuild}`);
    const r2 = spawnSync(cfg.goBuild, {
        cwd: cfg.appDir,
        stdio: "inherit",
        shell: true,
        env: goEnv(),
    });
    if (r2.status !== 0)
        throw new Error("Go build failed");
    console.log("✓ build done");
}
