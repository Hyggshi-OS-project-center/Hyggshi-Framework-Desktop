import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { loadConfig, goEnv } from "./config.mjs";
import { emitBindings } from "./ts-emitter.mjs";

// `hfd bindings [dir]` — chạy app Go ở codegen mode (HFD_CODEGEN=bindings go
// run .) để thu schema JSON từ stdout, rồi sinh bindings.d.ts + bindings.js.
export function runBindings(dir) {
	const cfg = loadConfig(dir);

	console.log(`▸ codegen bindings từ ${cfg.appDir}`);
	let schemaText;
	try {
		schemaText = execFileSync("go", ["run", "."], {
			cwd: cfg.appDir,
			encoding: "utf8",
			env: { ...goEnv(), HFD_CODEGEN: "bindings" },
			maxBuffer: 64 * 1024 * 1024,
		});
	} catch (err) {
		const stderr = err.stderr || "";
		throw new Error(`go run thất bại:\n${stderr.trim() || err.message}`);
	}

	const schemaJSON = schemaText.slice(schemaText.indexOf("{"));
	const schema = JSON.parse(schemaJSON);
	if (schema.error) throw new Error(`schema lỗi: ${schema.error}`);
	if (!Array.isArray(schema.bindings)) throw new Error("schema sai định dạng (thiếu bindings[])");

	const { dts, js } = emitBindings(schema);

	mkdirSync(cfg.bindingsDir, { recursive: true });
	writeFileSync(join(cfg.bindingsDir, "bindings.d.ts"), dts);
	writeFileSync(join(cfg.bindingsDir, "bindings.js"), js);

	const names = schema.bindings.map((b) => b.name).join(", ");
	console.log(`✓ sinh ${schema.bindings.length} bindings (${names}) → ${cfg.bindingsDir}`);
}
