// cmd-bindings.ts — `hfd bindings [dir]`
//
// Runs the Go app in codegen mode (HFD_CODEGEN=bindings go run .) to collect
// the schema JSON from stdout, then emits bindings.d.ts + bindings.js.
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { loadConfig, goEnv } from "./config.js";
import { emitBindings } from "./ts-emitter.js";
export async function runBindings(dir) {
    const cfg = loadConfig(dir);
    console.log(`▸ codegen bindings from ${cfg.appDir}`);
    let schemaText;
    try {
        schemaText = execFileSync("go", ["run", "."], {
            cwd: cfg.appDir,
            encoding: "utf8",
            env: { ...goEnv(), HFD_CODEGEN: "bindings" },
            maxBuffer: 64 * 1024 * 1024,
        });
    }
    catch (err) {
        const e = err;
        const stderr = e.stderr ?? "";
        throw new Error(`go run failed:\n${stderr.trim() || e.message}`);
    }
    const schemaJSON = schemaText.slice(schemaText.indexOf("{"));
    const schema = JSON.parse(schemaJSON);
    if (schema.error)
        throw new Error(`schema error: ${schema.error}`);
    if (!Array.isArray(schema.bindings))
        throw new Error("malformed schema (missing bindings[])");
    const { dts, js } = emitBindings(schema);
    mkdirSync(cfg.bindingsDir, { recursive: true });
    writeFileSync(join(cfg.bindingsDir, "bindings.d.ts"), dts);
    writeFileSync(join(cfg.bindingsDir, "bindings.js"), js);
    const names = schema.bindings.map((b) => b.name).join(", ");
    console.log(`✓ generated ${schema.bindings.length} bindings (${names}) → ${cfg.bindingsDir}`);
}
