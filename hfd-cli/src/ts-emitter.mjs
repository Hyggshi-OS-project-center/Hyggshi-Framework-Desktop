// ts-emitter: schema JSON (từ core/bindings_schema.go) → TypeScript.
//
// Chiến lược: pre-walk toàn graph để thu mọi named type (struct có tên hoặc
// type được đặt tên), declare 1 lần, rồi render biểu thức TS trỏ tới tên đó.
// Anonymous struct → inline object literal. Tên trùng giữa các package → suffix.

export function emitBindings(schema) {
	const named = new Map(); // key "pkg.Name" → node
	const tsNames = new Map(); // key → tên TS cuối (xử lý collision)

	for (const b of schema.bindings ?? []) {
		collectNamed(b.in, named);
		collectNamed(b.out, named);
	}

	// gán tên TS, suffix _2, _3… khi trùng
	const usedNames = new Set();
	for (const [key, node] of named) {
		let name = node.name;
		for (let i = 2; usedNames.has(name); i++) name = `${node.name}_${i}`;
		usedNames.add(name);
		tsNames.set(key, name);
	}

	const decls = [...named.keys()].map((key) =>
		declareType(named.get(key), tsNames.get(key), tsNames)
	);

	const handlerEntries = (schema.bindings ?? []).map(
		(b) => `\t${b.name}: [${tsOf(b.in, tsNames)}, ${tsOf(b.out, tsNames)}];`
	);

	const dts = [
		"// TỰ SINH bởi `hfd bindings` — KHÔNG sửa tay.",
		"// Source of truth: các hàm hfd.BindTyped trong Go.",
		"",
		...decls,
		"",
		"export interface HFDHandlers {",
		...handlerEntries,
		"}",
		"",
		"export declare function call<K extends keyof HFDHandlers>(",
		"\tname: K,",
		"\targs: HFDHandlers[K][0],",
		"): Promise<HFDHandlers[K][1]>;",
		"export declare function on(name: string, cb: (data: unknown) => void): () => void;",
		"export declare function off(name: string, cb: (data: unknown) => void): void;",
		"",
	].join("\n");

	const js = [
		"// TỰ SINH bởi `hfd bindings` — glue quanh window.hfd (runtime Go inject).",
		"if (typeof window !== \"undefined\") {",
		"\twindow.hfdAPI = {",
		"\t\tcall(name, args) { return window.hfd.call(name, args); },",
		"\t\ton(name, cb) { return window.hfd.on(name, cb); },",
		"\t\toff(name, cb) { window.hfd.off(name, cb); },",
		"\t};",
		"}",
		"export function call(name, args) {",
		"\treturn window.hfd.call(name, args);",
		"}",
		"export function on(name, cb) {",
		"\treturn window.hfd.on(name, cb);",
		"}",
		"export function off(name, cb) {",
		"\twindow.hfd.off(name, cb);",
		"}",
		"",
	].join("\n");

	return { dts, js };
}

// ---- collect: duyệt graph, thu mọi named type

function collectNamed(node, out) {
	if (!node) return;
	if (node.name) {
		const key = `${node.pkg || "<root>"}.${node.name}`;
		if (node.kind === "struct") {
			if (!out.has(key)) {
				out.set(key, node);
				for (const f of node.fields ?? []) collectNamed(f.type, out);
				return;
			}
		} else {
			out.set(key, node);
		}
	}
	switch (node.kind) {
		case "array":
		case "ptr":
			collectNamed(node.elem, out);
			break;
		case "map":
			collectNamed(node.key, out);
			collectNamed(node.val, out);
			break;
		case "struct": // anonymous struct
			for (const f of node.fields ?? []) collectNamed(f.type, out);
			break;
	}
}

// ---- declare: sinh "export interface X" hoặc "export type X"

function declareType(node, tsName, names) {
	if (node.kind === "struct") {
		const fields = (node.fields ?? [])
			.map((f) => `\t${f.name}: ${tsOf(f.type, names)};`)
			.join("\n");
		return `export interface ${tsName} {\n${fields || "\t// struct rỗng"}\n}`;
	}
	// type đặt tên trên primitive/slice/map → alias underlying
	return `export type ${tsName} = ${tsOf({ ...node, name: "", pkg: "" }, names)};`;
}

// ---- render biểu thức TS

function tsOf(node, names) {
	if (!node) return "unknown";
	if (node.name) {
		const key = `${node.pkg || "<root>"}.${node.name}`;
		return names.get(key) ?? node.name;
	}
	switch (node.kind) {
		case "array":
			return `${tsOf(node.elem, names)}[]`;
		case "ptr":
			return `${tsOf(node.elem, names)} | null`;
		case "map":
			return `Record<${node.key && node.key.kind === "string" ? "string" : tsOf(node.key, names)}, ${tsOf(node.val, names)}>`;
		case "struct": // anonymous
			return `{ ${(node.fields ?? [])
				.map((f) => `${f.name}: ${tsOf(f.type, names)}`)
				.join("; ")} }`;
		default:
			return { string: "string", number: "number", bool: "boolean" }[node.kind] ?? "unknown";
	}
}
