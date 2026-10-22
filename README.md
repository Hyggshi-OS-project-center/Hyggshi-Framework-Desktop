# HFD — Hyggshi Framework Desktop

Framework desktop tự viết 100% (không dùng Wails/Tauri/Electron):

- **Go** — runtime + native API (CGo, GTK3 + WebKitGTK trên Linux)
- **TypeScript** — CLI/devtools (`hfd-cli`, Node 0 dep)
- **Frontend tùy ý** — vanilla/React/Vue + Vite (HMR qua `hfd dev`)
- **Codegen type-safe** — `BindTyped[In,Out]` → sinh `bindings.d.ts`/`bindings.js`

## Trạng thái (v0.1.0 — Linux first)

| Phase | Nội dung | Trạng thái |
|---|---|---|
| 1 | Core CGo: GTK + WebKitGTK, IPC promise thật, scheme `hfd://app` | ✅ |
| 2 | CLI `hfd bindings/build/dev/new` + codegen type-safe | ✅ |
| 3 | `@hfd/react` (useHfdCall/useHfdEvent) + scaffold React/Vue | ✅ |
| 4 | Native API: file dialog, message box, clipboard, notify, title, menu bar, **tray SNI+DBusMenu** | ✅ (tray cần `-tags hfd_tray`) |
| 5 | Backend Windows/macOS | ⏸ scaffold + test plan: `docs/PHASE5-BACKENDS.md` |
| 6 | Security: allowlist origin 2 tầng + per-launch token + CSP | ✅ |

Chi tiết kiến trúc: `ARCHITECTURE.md`.

## Quickstart

```bash
# máy Linux dev: cần Go + Node + GTK3/WebKitGTK dev packages (apt)
git clone <repo> && cd HFD
npm i --prefix hfd-cli 2>/dev/null || true   # CLI 0-dep — bước này optional

cd examples/demo
node ../../hfd-cli/bin/hfd.mjs bindings   # codegen bindings.d.ts + bindings.js
node ../../hfd-cli/bin/hfd.mjs build      # go build → build/bin/hfd-demo
./build/bin/hfd-demo
```

Windows/macOS build hiện chưa bật (scaffold) — xem `docs/PHASE5-BACKENDS.md`.

## Lệnh CLI

| Lệnh | Tác dụng |
|---|---|
| `hfd new <tên> [--react|--vue]` | scaffold app mới vào `./<tên>` (go.mod + frontend + hfd.json) |
| `hfd bindings` | chạy `HFD_CODEGEN=bindings go run .` → schema JSON → `bindings.d.ts` + `bindings.js` vào `bindingsDir` |
| `hfd build` | `hfd bindings` + `go build` (lấy `goBuild` từ hfd.json) |
| `hfd dev` | chạy dev server (vite/npm) + build Go với env `HFD_DEV_URL` — HMR không restart app |

## API chính (Go)

```go
hfd.Run(hfd.Options{
    Title:  "Demo", Width: 880, Height: 700,
    Assets: assets,                         // go:embed embed.FS
    AllowOrigins: []string{"hfd://app/*"},  // nil → mặc định hfd://app/* (+devURL)
    CSP:    "default-src hfd://app/",       // inject <meta> DOCUMENT_START
    Menu:   []hfd.MenuItem{{Label:"Hỗ trợ", Action:"about"}},
    Debug:  true,                           // inspector + log IPC
})

hfd.BindTyped("Greet", func(ctx,hfd.Context, name,string) (string, error) { ... })
hfd.Emit("tick", map[string]any{"count": n})

hfd.SetClipboard / GetClipboard
hfd.OpenFileDialog / SaveFileDialog / ShowMessage
hfd.Notify("tiêu đề", "nội dung")
hfd.SetTitle / CloseWindow
hfd.SetTray(hfd.TrayOptions{Tooltip:"…", Items: []hfd.TrayItem{{Label:"Thoát", Action:"quit"}}})
```

Phía JS: `api().call("Greet",{name})` trả **Promise thật**; `api().on("tick", fn)`
nhận event Go→JS. Codegen cho type-safe: `bindings.d.ts`.

## Cấu trúc repo

```
core/           Go runtime + backend GTK/WebKitGTK (0 external dep mặc định)
  runtime/      hfd.js — bridge inject DOCUMENT_START (0 dep)
hfd-cli/src/    CLI Node 0-dep (bindings/build/dev/new)
packages/react/ @hfd/react hooks
examples/demo/  demo vanilla đầy đủ tính năng (security, menu, tray, native API)
examples/react-demo/ demo React (vite HMR)
docs/           test plan Phase 5 (Windows/macOS)
```

## Nền tảng
- ✅ Linux (GTK3 + WebKitGTK 4.1) — phát triển trên Ubuntu GNOME
- ⏸ Windows/macOS — scaffold: `core/webview_*` + `docs/PHASE5-BACKENDS.md`
