// app.js — gọi qua window.hfdAPI (được bindings.js codegen set, có types
// ở bindings.d.ts — xem frontend/dist/hfd/).
const $ = (id) => document.getElementById(id);
const api = () => window.hfdAPI;

const show = (el, data) => {
	el.textContent = typeof data === "string" ? data : JSON.stringify(data, null, 2);
};

// ---- auto-probe: chạy ngay khi load — kiểm chứng chuỗi runtime inject →
// bindings codegen → IPC native reply mà không cần bấm gì cả.
const boot = $("boot");
boot.textContent = "…calling SysInfo over the native-reply IPC…";

api()
	.call("SysInfo", {})
	.then((info) => {
		boot.textContent =
			`✓ auto-probe OK — ${info.hostname} • ${info.os}/${info.arch} • ` +
			`${info.cpus} CPUs • Go ${info.goVersion} • [${info.tags.join(", ")}]`;
	})
	.catch((e) => {
		boot.textContent = "✗ auto-probe FAILED: " + e.message;
		console.error(e);
	});

// ---- actions

$("greet").addEventListener("click", async () => {
	try {
		const res = await api().call("Greet", { name: $("name").value });
		show($("greet-result"), res.message);
	} catch (e) {
		show($("greet-result"), "✗ " + e.message);
	}
});

$("sysinfo").addEventListener("click", async () => {
	try {
		show($("sysinfo-result"), await api().call("SysInfo", {}));
	} catch (e) {
		show($("sysinfo-result"), "✗ " + e.message);
	}
});

let ticksOn = false;

$("tick").addEventListener("click", async () => {
	if (ticksOn) return;
	ticksOn = true;
	show($("tick-result"), "ticking…");
	api().on("tick", (t) => show($("tick-result"), `event from Go — tick #${t.count}`));
	api().on("tick-done", (t) => {
		show($("tick-result"), `done — ${t.note}`);
		ticksOn = false;
	});
	await api().call("StartTick", { times: 3 }); // times từ JS → Go typed!
});

// ---- Phase 4: native API demos

const nat = $("native-result");

$("pick").addEventListener("click", async () => {
	const res = await api().call("PickFile", { title: "Pick a file (served by Go)" });
	show(nat, res.canceled ? "canceled" : `file picked via GTK: ${res.path}`);
});

$("save").addEventListener("click", async () => {
	const res = await api().call("SaveText", { name: "hfd-note.txt", text: new Date().toString() });
	res.canceled ? show(nat, "save canceled") : show(nat, `saved ${res.bytes} bytes → ${res.path}`);
});

$("confirm").addEventListener("click", async () => {
	const res = await api().call("AskConfirm", {});
	show(nat, res.text);
});

$("clip-set").addEventListener("click", async () => {
	await api().call("ClipboardSet", { text: "hfd clipboard ✓ — copied straight from the Go runtime" });
	show(nat, "clipboard set ✓ — try Ctrl+V somewhere else");
});

$("clip-get").addEventListener("click", async () => {
	const res = await api().call("ClipboardGet", {});
	show(nat, res.text === "" ? "clipboard is empty" : `clipboard: ${res.text}`);
});

$("notify").addEventListener("click", async () => {
	const res = await api().call("SendNotification", { title: "HFD", body: "Notification fired from the Go runtime!" });
	res.sent ? show(nat, "notification ✓ — check your desktop") : show(nat, "✗ " + res.err);
});

// ---- tray (cần binary build với -tags hfd_tray + shell hỗ trợ SNI)
api().on("tray", ({ action }) => {
	console.log("[hfd] tray event:", action);
	if (action === "__activate") { show($("menu-result"), "tray icon clicked"); return; }
	if (action === "tray-tick") { $("tick").click(); show($("menu-result"), "tray: tick fired"); return; }
	if (action === "tray-about") { show($("menu-result"), "HFD — the tray menu of Hyggshi Framework Desktop"); return; }
	if (action === "tray-quit") { show($("menu-result"), "tray: quit — hfd.CloseWindow()"); return; }
	show($("menu-result"), "tray action: " + action);
});

// ---- menu bar
api().on("menu", ({ action }) => {
	if (action === "open-file") { $("pick").click(); return; }
	if (action === "notify") { $("notify").click(); return; }
	if (action === "about") {
		show($("menu-result"), "HFD — Hyggshi Framework Desktop\nGo + GTK/WebKitGTK, written 100% from scratch");
	} else {
		show($("menu-result"), "menu action: " + action);
	}
});
