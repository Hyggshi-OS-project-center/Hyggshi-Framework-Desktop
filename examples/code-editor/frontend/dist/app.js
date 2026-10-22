// app.js — HFD Code Editor (vanilla, no build step)
// Calls Go through generated typed bindings (window.hfdAPI, see hfd/).
const $ = (id) => document.getElementById(id);
// hfdAPI is injected by the Go runtime via user script. When absent
// (static preview in a plain browser) the app degrades to a plain editor.
const api = () => window.hfdAPI;

const editor = $("editor"), hl = $("hl-code"), gutter = $("gutter");

const state = { path: "", dirty: false };
const HAS_HFD = !!api();

const normKey = (e) => (e.ctrlKey || e.metaKey) ? e.key.toLowerCase() : null;

// ---- syntax highlighting ----------------------------------------------------
// One pass-regex per language, applied to ESCAPED html. Heuristic, but plenty
// fast for the 4 MB cap we enforce in Go.

const LANGS = {
	go: { name: "Go", ext: ["go"],
		re: /("(?:[^"\\]|\\.)*"|`[^`]*`|'(?:[^'\\]|\\.)*')|(\/\/[^\n]*)|(\/\*[\s\S]*?\*\/)|(\b(?:package|import|func|return|type|struct|if|else|for|range|switch|case|default|var|const|nil|true|false|map|chan|defer|go|select|break|continue|interface)\b)|(\b(?:[A-Z][A-Za-z0-9_]*)\b)|(\b(?:0x[0-9a-f]+|\d+(?:\.\d+)?)\b)/g },
	js: { name: "JavaScript", ext: ["js", "mjs", "cjs"],
		re: /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)|(\/\/[^\n]*)|(\/\*[\s\S]*?\*\/)|(\b(?:const|let|var|function|return|if|else|for|while|switch|case|break|continue|new|class|extends|import|export|from|default|try|catch|finally|throw|await|async|this|null|undefined|true|false|typeof|instanceof|delete|in|of)\b)|(\b(\d+(?:\.\d+)?)\b)/g },
	ts: { name: "TypeScript", ext: ["ts", "tsx"],
		re: /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)|(\/\/[^\n]*)|(\/\*[\s\S]*?\*\/)|(\b(?:const|let|var|function|return|if|else|for|while|switch|case|break|continue|new|class|extends|interface|type|implements|enum|namespace|declare|readonly|public|private|protected|import|export|from|default|try|catch|finally|throw|await|async|this|null|undefined|true|false|typeof|instanceof|keyof|as|in|of)\b)|(\b(?:\d+(?:\.\d+)?)\b)/g },
	json: { name: "JSON", ext: ["json"],
		re: /("(?:[^"\\]|\\.)*")(\s*:)?|(\b(?:true|false|null)\b)|(-?\d+(?:\.\d+(?:[eE][+-]?\d+)?)?)/g },
	css: { name: "CSS", ext: ["css"],
		re: /(\/\*[\s\S]*?\*\/)|("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(#[0-9a-fA-F]{3,8}\b|@[\w-]+)|([.-]?[a-zA-Z-]+)(?=\s*\{)|(\b\d+(?:\.\d+)?(?:px|em|rem|vh|vw|%|s|ms)?\b)/g },
	html: { name: "HTML", ext: ["html", "htm", "svg"], re: null }, // handled inline
};

function langFor(path) {
	const ext = (path.split(".").pop() || "").toLowerCase();
	for (const key of Object.keys(LANGS)) {
		if (LANGS[key].ext.includes(ext)) return key;
	}
	return "plain";
}
const LANG_MAP = new Map();
for (const [key, L] of Object.entries(LANGS)) {
	for (const e of L.ext) LANG_MAP.set(e, key);
}

function langName(kind) {
	return kind === "plain" ? "plain text" : LANGS[kind].name;
}

const esc = (s) => s.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");

function highlightCSSishPlus(src, re, classes) {
	return esc(src).replace(re, (m, ...groups) => {
		const i = groups.slice(0, classes.length).findIndex((v) => v !== undefined);
		return i >= 0 ? `<span class="${classes[i]}">${m}</span>` : m;
	});
}

function highlight(src, kind) {
	if (kind === "html") {
		// token colors for tags/attrs around an escaped stream
		return esc(src)
			.replace(/(&lt;\/?)([\w-]+)([^&]*?)(\/?&gt;)/g,
				(m, o, tag, attrs, c) =>
					`${o}<span class="tk-f">${tag}</span>` +
					attrs.replace(/([\w-]+)(=)("(?:[^"\\]|\\.)*")/g,
						(_m, a, eq, v) => `<span class="tk-t">${a}</span>${eq}<span class="tk-s">${v}</span>`) +
					c)
			.replace(/(&lt;!--[\s\S]*?--&gt;)/g, `<span class="tk-c">$1</span>`);
	}
	if (kind === "go") return highlightCSSishPlus(src, LANGS.go.re,
		["tk-s", "tk-c", "tk-c", "tk-k", "tk-t", "tk-n"]);
	if (kind === "js") return highlightCSSishPlus(src, LANGS.js.re,
		["tk-s", "tk-c", "tk-c", "tk-k", "tk-n"]);
	if (kind === "ts") return highlightCSSishPlus(src, LANGS.ts.re,
		["tk-s", "tk-c", "tk-c", "tk-k", "tk-n"]);
	if (kind === "json") {
		return esc(src).replace(LANGS.json.re, (m, str, colon, kw, num) => {
			if (str !== undefined) return `<span class="${colon ? "tk-p" : "tk-s"}">${str}</span>${colon ? colon : ""}`;
			if (kw) return `<span class="tk-k">${kw}</span>`;
			if (num) return `<span class="tk-n">${num}</span>`;
			return m;
		});
	}
	if (kind === "css") {
		return esc(src).replace(LANGS.css.re, (m, com, str, hexat, sel, num) => {
			if (com) return `<span class="tk-c">${com}</span>`;
			if (str) return `<span class="tk-s">${str}</span>`;
			if (hexat) return `<span class="tk-n">${hexat}</span>`;
			if (sel) return `<span class="tk-f">${sel}</span>`;
			if (num) return `<span class="tk-n">${num}</span>`;
			return m;
		});
	}
	return esc(src); // plain text
}

// ---- rendering: keep gutter + highlight in sync with the textarea ----------

function refresh(force) {
	const src = editor.value;

	// line numbers
	const lines = src.split("\n").length;
	if (force || gutter.childElementCount !== lines) {
		let out = "";
		for (let i = 1; i <= lines; i++) out += i + "\n";
		gutter.textContent = out;
	}

	// highlight (skip huge payloads to stay smooth)
	const kind = langFor(state.path);
	if (src.length <= 200_000) {
		hl.innerHTML = highlight(src, kind) + "\n";
	} else {
		hl.textContent = ""; // cheap fallback
	}

	updateStatus(src, kind);
}

function updateStatus(src, kind) {
	const lines = src.length ? src.split("\n").length : 1;
	$("stats").textContent =
		`${lines} line${lines === 1 ? "" : "s"} · ${new Blob([src]).size} B`;
	$("lang").textContent = langName(kind);
	const upto = src.slice(0, editor.selectionStart);
	const nl = upto.lastIndexOf("\n");
	const col = upto.length - nl;
	$("pos").textContent = `Ln ${nl < 0 ? 1 : upto.slice(0, nl + 1).split("\n").length}, Col ${col + 1}`;
}

// ---- dirty tracking + title -------------------------------------------------

const toolbarTitle = document.querySelector("#toolbar strong");

function setDirty(v) {
	state.dirty = v;
	$("dirty-dot").hidden = !v;
	toolbarTitle.textContent =
		(state.path ? state.path.replace(/^.*\//, "") : "untitled.txt") + " — HFD Code (demo)" + (v ? " ●" : "");
}

function setFile(path) {
	state.path = path;
	$("file-name").value = path || "untitled.txt";
	$("file-name").title = path || "current file";
	setDirty(false);
	refresh(true);
}

// ---- Go interactions --------------------------------------------------------

async function openViaDialog() {
	if (!HAS_HFD) { note("static preview — no HFD runtime", "err"); return; }
	try {
		const res = await api().call("OpenFileDialog", {});
		if (res.canceled) return;
		editor.value = res.content;
		setFile(res.path);
		note(`opened ${res.path} ✓`, "ok");
	} catch (e) {
		note("✗ " + e.message, "err");
	}
}

async function saveFile(saveAs) {
	if (!HAS_HFD) { note("static preview — no HFD runtime", "err"); return; }
	const text = editor.value;
	try {
		const res = await api().call("SaveFile", { path: saveAs ? "" : state.path, text });
		setFile(res.path);
		note(`saved ${res.bytes} bytes → ${res.path} ✓`, "ok");
	} catch (e) {
		note("✗ " + e.message, "err");
	}
}

function newFile() {
	editor.value = "";
	state.path = "";
	if (HAS_HFD) api().call("SetTitle", { title: "untitled.txt — HFD Code" });
	setFile("");
	note("new file", "ok");
}

const noteEl = $("save-note");
let noteTimer = 0;
function note(msg, cls) {
	noteEl.textContent = msg;
	noteEl.className = cls || "";
	clearTimeout(noteTimer);
	noteTimer = setTimeout(() => { noteEl.textContent = ""; noteEl.className = ""; }, 4000);
}

// ---- events -----------------------------------------------------------------

editor.addEventListener("input", () => { setDirty(true); refresh(); });
editor.addEventListener("scroll", () => {
	// push both replicas with the textarea's scroll
	hl.scrollTop = editor.scrollTop;
	hl.scrollLeft = editor.scrollLeft;
	gutter.scrollTop = editor.scrollTop;
});
editor.addEventListener("keyup", () => refresh());
editor.addEventListener("click", () => refresh());
editor.addEventListener("keydown", (e) => {
	const k = normKey(e);
	if (k === "s") { e.preventDefault(); saveFile(e.shiftKey); }
	else if (k === "o") { e.preventDefault(); openViaDialog(); }
	else if (k === "n") { e.preventDefault(); newFile(); }
	else if (e.key === "Tab") {
		// insert a real tab
		e.preventDefault();
		const s = editor.selectionStart, t = editor.selectionEnd;
		editor.setRangeText("\t", s, t, "end");
		setDirty(true);
		refresh();
	}
});

window.addEventListener("dragover", (e) => e.preventDefault());
window.addEventListener("drop", (e) => {
	e.preventDefault();
	const f = e.dataTransfer?.files?.[0];
	if (!f) return;
	f.text().then((content) => {
		editor.value = content;
		setFile("dropped:" + f.name);
		note(`loaded ${f.name} from drop ✓`, "ok");
	});
});

// toolbar
$("btn-new").addEventListener("click", newFile);
$("btn-open").addEventListener("click", openViaDialog);
$("btn-save").addEventListener("click", () => saveFile(false));

// menu bar (Go → JS)
if (HAS_HFD) {
	api().on("menu", ({ action }) => {
		if (action === "file-new") newFile();
		else if (action === "file-open") openViaDialog();
		else if (action === "file-save") saveFile(false);
		else if (action === "file-saveas") saveFile(true);
		else if (action === "help-about") api().call("AboutApp");
	});
}

// boot: put a starter snippet in the buffer so the demo shows highlighting
// immediately (only in the pristine state — never overwrite an open file)
if (!editor.value && !state.path) {
	editor.value = `// HFD Code Editor demo
// Open a real file with Ctrl+O, save with Ctrl+S.

package main

import "fmt"

func main() {
	fmt.Println("Hello from Hyggshi Framework Desktop!")
}
`;
}
refresh(true);
