# HFD — Hyggshi Framework Desktop

A desktop framework written 100% in-house (no Wails/Tauri/Electron):

- **Go** — runtime + native API (CGo, GTK3 + WebKitGTK on Linux)
- **TypeScript** — CLI/devtools (`hfd-cli`, Node 0-dep)
- **Any frontend** — vanilla/React/Vue + Vite (HMR via `hfd dev`)
- **Type-safe codegen** — `BindTyped[In,Out]` → generates `bindings.d.ts`/`bindings.js`

## Status (v0.1.0 — Linux first)

| Phase | Scope | Status |
|---|---|---|
| 1 | Core CGo: GTK + WebKitGTK, true-promise IPC, `hfd://app` scheme | ✅ |
| 2 | CLI `hfd bindings/build/dev/new` + type-safe codegen | ✅ |
| 3 | `@hfd/react` (useHfdCall/useHfdEvent) + React/Vue scaffolds | ✅ |
| 4 | Native API: file dialogs, message box, clipboard, notify, title, menu bar, **tray (SNI + DBusMenu)** | ✅ (tray requires `-tags hfd_tray`) |
| 5 | Windows/macOS backends | ⏸ scaffold + test plan: `docs/PHASE5-BACKENDS.md` |
| 6 | Security: 2-layer origin allowlist + per-launch token + CSP | ✅ |

Full architecture details: `ARCHITECTURE.md`.

## Quickstart

```bash
# Linux dev machine: Go + Node + GTK3/WebKitGTK dev packages (apt)
git clone <repo> && cd HFD
npm i --prefix hfd-cli 2>/dev/null || true   # CLI is 0-dep — this step is optional

cd examples/demo
node ../../hfd-cli/bin/hfd.mjs bindings   # generates bindings.d.ts + bindings.js
node ../../hfd-cli/bin/hfd.mjs build      # go build → build/bin/hfd-demo
./build/bin/hfd-demo
```

Windows/macOS builds are not enabled yet (scaffold only) — see `docs/PHASE5-BACKENDS.md`.

## CLI commands

| Command | Purpose |
|---|---|
| `hfd new <name> [--react|--vue]` | scaffold a new app into `./<name>` (go.mod + frontend + hfd.json) |
| `hfd bindings` | runs `HFD_CODEGEN=bindings go run .` → schema JSON → `bindings.d.ts` + `bindings.js` into `bindingsDir` |
| `hfd build` | `hfd bindings` + `go build` (uses `goBuild` from hfd.json) |
| `hfd dev` | starts the dev server (vite/npm) + builds Go with `HFD_DEV_URL` — HMR without app restart |

## Main API (Go)

```go
hfd.Run(hfd.Options{
    Title:  "Demo", Width: 880, Height: 700,
    Assets: assets,                         // go:embed embed.FS
    AllowOrigins: []string{"hfd://app/*"},  // nil → defaults to hfd://app/* (+devURL)
    CSP:    "default-src hfd://app/",       // injects <meta> at DOCUMENT_START
    Menu:   []hfd.MenuItem{{Label: "Support", Action: "about"}},
    Debug:  true,                           // inspector + IPC logging
})

hfd.BindTyped("Greet", func(ctx hfd.Context, name string) (string, error) { ... })
hfd.Emit("tick", map[string]any{"count": n})

hfd.SetClipboard / GetClipboard
hfd.OpenFileDialog / SaveFileDialog / ShowMessage
hfd.Notify("title", "body")
hfd.SetTitle / CloseWindow
hfd.SetTray(hfd.TrayOptions{Tooltip: "…", Items: []hfd.TrayItem{{Label: "Quit", Action: "quit"}}})
```

JS side: `api().call("Greet", {name})` returns a **real Promise**;
`api().on("tick", fn)` receives Go→JS events. Codegen provides type
safety via `bindings.d.ts`.

## Repo layout

```
core/           Go runtime + GTK/WebKitGTK backend (0 external deps by default)
  runtime/      hfd.js — bridge injected at DOCUMENT_START (0-dep)
hfd-cli/src/    Node CLI, 0-dep (bindings/build/dev/new)
packages/react/ @hfd/react hooks
examples/demo/  vanilla demo with full feature coverage (security, menu, tray, native API)
examples/react-demo/ React demo (vite HMR)
docs/           Phase 5 test plan (Windows/macOS)
```

## Platforms

- ✅ Linux (GTK3 + WebKitGTK 4.1) — developed on Ubuntu GNOME
- ⏸ Windows/macOS — scaffold ready: `core/webview_*` + `docs/PHASE5-BACKENDS.md`
