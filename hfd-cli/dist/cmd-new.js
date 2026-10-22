// cmd-new.ts — `hfd new <name> [--template react|vue] [--core path]`
//
// Scaffolds a fresh HFD project (react or vue + vite). The bundled template
// uses BindTyped + @hfd/react; go.mod replaces the core by a relative path
// configured with --core.
import { mkdirSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
function reactTemplates(name, moduleName, coreRel, reactRel) {
    const T = {};
    T["go.mod"] = `module ${moduleName}

go 1.27

require github.com/hyggshi/hfd v0.1.0

replace github.com/hyggshi/hfd => ${coreRel}
`;
    T["main.go"] = `package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"runtime"

	"github.com/hyggshi/hfd"
)

//go:embed all:frontend/dist
var distFS embed.FS

type GreetArgs struct {
	Name string \`json:"name"\`
}

type GreetResult struct {
	Message string \`json:"message"\`
}

type SysInfoArgs struct{}

type SysInfoResult struct {
	Hostname string \`json:"hostname"\`
	OS       string \`json:"os"\`
	Arch     string \`json:"arch"\`
	CPUs     int    \`json:"cpus"\`
	GoVer    string \`json:"goVersion"\`
}

type App struct{}

func (a *App) Greet(args GreetArgs) (GreetResult, error) {
	if args.Name == "" {
		args.Name = "there"
	}
	return GreetResult{Message: "Hello " + args.Name + " from HFD!"}, nil
}

func (a *App) SysInfo(_ SysInfoArgs) (SysInfoResult, error) {
	hostname, _ := os.Hostname()
	return SysInfoResult{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
		GoVer:    runtime.Version(),
	}, nil
}

func main() {
	assets, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	app := &App{}
	hfd.BindTyped("Greet", app.Greet)
	hfd.BindTyped("SysInfo", app.SysInfo)

	if err := hfd.Run(hfd.Options{
		Title:  "${name}",
		Width:  900,
		Height: 640,
		Assets: assets,
		Debug:  true,
	}); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
`;
    T["hfd.json"] = `{
	"app": { "dir": "." },
	"bindingsDir": "frontend/src/hfd",
	"goBuild": "go build -o build/bin/${name} .",
	"frontend": {
		"dir": "frontend",
		"dev": "npm run dev",
		"build": "npm run build",
		"port": 5173
	}
}
`;
    T["frontend/package.json"] = `{
  "name": "${name}-frontend",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build"
  },
  "dependencies": {
    "@hfd/react": "${reactRel}",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@types/react": "^18.3.3",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.1",
    "typescript": "^5.5.0",
    "vite": "^5.4.0"
  }
}
`;
    T["frontend/vite.config.ts"] = `import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// Build output ./dist — go:embed all:frontend/dist embeds it into the binary.
export default defineConfig({
	plugins: [react()],
	build: { outDir: "dist" },
});
`;
    T["frontend/tsconfig.json"] = `{
	"compilerOptions": {
		"target": "ES2022",
		"module": "ESNext",
		"moduleResolution": "bundler",
		"jsx": "react-jsx",
		"strict": true,
		"skipLibCheck": true,
		"noEmit": true,
		"lib": ["ES2022", "DOM", "DOM.Iterable"]
	},
	"include": ["src"]
}
`;
    T["frontend/index.html"] = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
</head>
<body>
	<div id="root"></div>
	<script type="module" src="/src/main.tsx"></script>
</body>
</html>
`;
    T["frontend/src/main.tsx"] = `import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
`;
    T["frontend/src/App.tsx"] = `import { useState } from "react";
import { call, useHfdCall } from "@hfd/react";

interface SysInfo {
	hostname: string;
	os: string;
	arch: string;
	cpus: number;
	goVersion: string;
}

export default function App() {
	const sys = useHfdCall<null, SysInfo>("SysInfo"); // default args → Go receives "null"
	const [name, setName] = useState("Hyggshi");
	const [greeted, setGreeted] = useState("");

	const greet = async () => {
		const res = (await call("Greet", { name })) as { message: string };
		setGreeted(res.message);
	};

	return (
		<main className="app">
			<h1>${name} <span className="tag">HFD + React + Vite</span></h1>
			<p className="sub">useHfdCall calls SysInfo on mount — over native IPC.</p>

			<section className="card">
				{sys.loading && <p>loading SysInfo…</p>}
				{sys.error && <p className="err">SysInfo failed: {String(sys.error.message)}</p>}
				{sys.data && (
					<pre className="output">
	{sys.data.hostname} • {sys.data.os}/{sys.data.arch} • {sys.data.cpus} CPUs • {sys.data.goVersion}
					</pre>
				)}
			</section>

			<section className="card">
				<input value={name} onChange={(e) => setName(e.target.value)} />
				<button onClick={greet}>Greet</button>
				{greeted && <div className="output">{greeted}</div>}
			</section>
		</main>
	);
}
`;
    T["frontend/src/styles.css"] = `:root { color-scheme: dark; }
body {
	margin: 0;
	font: 15px/1.55 system-ui, sans-serif;
	background: #16181d;
	color: #e8e9ec;
}
.app { max-width: 640px; margin: 0 auto; padding: 32px 20px; }
h1 { font-size: 22px; }
.tag { font-size: 11px; color: #8ecdb8; border: 1px solid #2f5c4b; padding: 2px 8px; border-radius: 999px; }
.sub { color: #9aa1ad; }
.card {
	background: #1e2127; border: 1px solid #33373f;
	border-radius: 10px; padding: 16px; margin-bottom: 14px;
}
.output, .err { background: #14161a; border: 1px solid #2c2f36; border-radius: 6px; padding: 10px 12px; font-family: ui-monospace, monospace; font-size: 12.5px; }
.err { color: #ff8e8e; }
input {
	background: #24272e; border: 1px solid #3a3f47; color: #e8e9ec;
	border-radius: 6px; padding: 7px 10px; font: inherit;
}
button {
	background: #2d6fb8; border: 0; color: #fff;
	border-radius: 6px; padding: 7px 14px; font: inherit; cursor: pointer;
}
`;
    return T;
}
function vueTemplates(name, moduleName, coreRel) {
    const T = {};
    // Go main.go, hfd.json, go.mod shared with react (same paths)
    const base = reactTemplates(name, moduleName, coreRel, "../packages/react");
    for (const k of ["go.mod", "main.go", "hfd.json"]) {
        T[k] = base[k] ?? "";
    }
    T["frontend/package.json"] = `{
  "name": "${name}-frontend",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build"
  },
  "dependencies": {
    "vue": "^3.4.31"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.0.5",
    "vite": "^5.4.0"
  }
}
`;
    T["frontend/vite.config.js"] = `import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [vue()],
	build: { outDir: "dist" },
});
`;
    T["frontend/index.html"] = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8" /></head>
<body>
	<div id="app"></div>
	<script type="module" src="/src/main.js"></script>
</body>
</html>
`;
    T["frontend/src/main.js"] = `import { createApp } from "vue";
import App from "./App.vue";

createApp(App).mount("#app");
`;
    T["frontend/src/App.vue"] = `<script setup>
import { ref } from "vue";

const sys = ref(null);
const greeted = ref("");
const name = ref("Hyggshi");

// window.hfd is injected automatically by the Go runtime.
window.hfd.call("SysInfo", {}).then((i) => { sys.value = i; });

const greet = async () => {
	const res = await window.hfd.call("Greet", { name: name.value });
	greeted.value = res.message;
};
</script>

<template>
	<main class="app">
		<h1>${name} <span class="tag">HFD + Vue + Vite</span></h1>
		<section class="card">
			<pre class="output" v-if="sys">{{ sys.hostname }} • {{ sys.os }}/{{ sys.arch }} • {{ sys.cpus }} CPUs</pre>
		</section>
		<section class="card">
			<input v-model="name" />
			<button @click="greet">Greet</button>
			<div class="output" v-if="greeted">{{ greeted }}</div>
		</section>
	</main>
</template>

<style>
:root { color-scheme: dark; }
body { margin: 0; font: 15px/1.55 system-ui, sans-serif; background: #16181d; color: #e8e9ec; }
.app { max-width: 640px; margin: 0 auto; padding: 32px 20px; }
h1 { font-size: 22px; }
.tag { font-size: 11px; color: #8ecdb8; border: 1px solid #2f5c4b; padding: 2px 8px; border-radius: 999px; }
.card { background: #1e2127; border: 1px solid #33373f; border-radius: 10px; padding: 16px; margin-bottom: 14px; }
.output { background: #14161a; border: 1px solid #2c2f36; border-radius: 6px; padding: 10px 12px; font-family: ui-monospace, monospace; font-size: 12.5px; }
input { background: #24272e; border: 1px solid #3a3f47; color: #e8e9ec; border-radius: 6px; padding: 7px 10px; font: inherit; }
button { background: #2d6fb8; border: 0; color: #fff; border-radius: 6px; padding: 7px 14px; font: inherit; cursor: pointer; }
</style>
`;
    return T;
}
function safeModule(name) {
    return name.replace(/[^a-zA-Z0-9_-]/g, "").replace(/^_+/, "") || "hfdapp";
}
export async function runNew(name, opts = {}) {
    const { template = "react", corePath = "../../core" } = opts;
    const moduleName = safeModule(name);
    const target = resolve(process.cwd(), name);
    if (template === "react") {
        writeFiles(target, reactTemplates(name, "hfd/" + moduleName, corePath, "../../../packages/react"));
    }
    else if (template === "vue") {
        writeFiles(target, vueTemplates(name, "hfd/" + moduleName, corePath));
    }
    else {
        throw new Error(`unknown template: ${template} (react or vue)`);
    }
    console.log(`✓ scaffold ${template} app → ${target}
Next steps:
  cd ${name}
  hfd bindings          # generate frontend/src/hfd/bindings.* from BindTyped
  (cd frontend && npm install && npm run build)
  hfd dev               # or: hfd build && build/bin/${name}
`);
}
function writeFiles(root, files) {
    for (const [rel, content] of Object.entries(files)) {
        const path = join(root, rel);
        mkdirSync(join(path, ".."), { recursive: true });
        writeFileSync(path, content);
    }
    // go:embed all:frontend/dist needs the directory to exist up front (even
    // before the first `npm run build` of the frontend).
    mkdirSync(join(root, "frontend", "dist"), { recursive: true });
    writeFileSync(join(root, "frontend", "dist", ".gitkeep"), "");
}
