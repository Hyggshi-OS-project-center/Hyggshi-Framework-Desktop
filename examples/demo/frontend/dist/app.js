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
boot.textContent = "…đang gọi SysInfo qua IPC native reply…";

api()
	.call("SysInfo", {})
	.then((info) => {
		boot.textContent =
			`✓ auto-probe OK — ${info.hostname} • ${info.os}/${info.arch} • ` +
			`${info.cpus} CPUs • Go ${info.goVersion} • [${info.tags.join(", ")}]`;
	})
	.catch((e) => {
		boot.textContent = "✗ auto-probe LỖI: " + e.message;
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
	show($("tick-result"), "đang tick…");
	api().on("tick", (t) => show($("tick-result"), `event từ Go — tick #${t.count}`));
	api().on("tick-done", (t) => {
		show($("tick-result"), `xong — ${t.note}`);
		ticksOn = false;
	});
	await api().call("StartTick", { times: 3 }); // times từ JS → Go typed!
});

// ---- Phase 4: native API demos

const nat = $("native-result");

$("pick").addEventListener("click", async () => {
	const res = await api().call("PickFile", { title: "Chọn file bên Go nhé" });
	show(nat, res.canceled ? "đã hủy gọi menu" : `file được chọn từ GTK: ${res.path}`);
});

$("save").addEventListener("click", async () => {
	const res = await api().call("SaveText", { name: "hfd-note.txt", text: new Date().toString() });
	res.canceled ? show(nat, "hủy lưu") : show(nat,`lưu được ${res.bytes} bytes → ${res.path}`);
});

$("confirm").addEventListener("click", async () => {
	const res = await api().call("AskConfirm", {});
	show(nat, res.text);
});

$("clip-set").addEventListener("click", async () => {
	await api().call("ClipboardSet", { text: "hfd clipboard ✓ — copy tự nhiên như NT" });
	show(nat, "clipboard set ✓ — thử Ctrl+V ở chỗ khác");
});

$("clip-get").addEventListener("click", async () => {
	const res = await api().call("ClipboardGet", {});
	show(nat, res.text === "" ? "clipboard rỗng" : `clipboard: ${res.text}`);
});

$("notify").addEventListener("click", async () => {
	const res = await api().call("SendNotification", { title: "HFD", body: "Notification bắn từ Go runtime!" });
	res.sent ? show(nat, "notification ✓ — nhìn trên màn hình") : show(nat, "✗ " + res.err);
});

// ---- tray (cần binary build với -tags hfd_tray + shell hỗ trợ SNI)
api().on("tray", ({ action }) => {
	console.log("[hfd] tray event:", action);
	if (action === "__activate") { show($("menu-result"), "tray icon clicked"); return; }
	if (action === "tray-tick") { $("tick").click(); show($("menu-result"), "tray: tick đã bắn"); return; }
	if (action === "tray-about") { show($("menu-result"), "HFD — tray menu của riêng Hyggshi Framework Desktop"); return; }
	if (action === "tray-quit") { show($("menu-result"), "tray: quit — hfd.CloseWindow()"); return; }
	show($("menu-result"), "tray action: " + action);
});

// ---- menu bar
api().on("menu", ({ action }) => {
	if (action === "open-file") { $("pick").click(); return; }
	if (action === "notify") { $("notify").click(); return; }
	if (action === "about") {
		show($("menu-result"), "HFD — Hyggshi Framework Desktop\nGo + GTK/WebKitGTK, tự viết 100%");
	} else {
		show($("menu-result"), "menu action: " + action);
	}
});
