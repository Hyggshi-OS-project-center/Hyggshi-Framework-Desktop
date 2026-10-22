import { runBindings } from "./cmd-bindings.mjs";
import { runBuild } from "./cmd-build.mjs";
import { runDev } from "./cmd-dev.mjs";
import { runNew } from "./cmd-new.mjs";
import { runDoctor } from "./cmd-doctor.mjs";
import { runPackage } from "./cmd-package.mjs";

const HELP = `
hfd — CLI for the Hyggshi Framework Desktop

Commands:
  hfd bindings [dir]   Generate type-safe TS bindings from Go (BindTyped),
                       reads the schema via 'HFD_CODEGEN=bindings go run .'
  hfd build [dir]      Build the frontend, then the Go binary → build/bin/
  hfd dev    [dir]     Run dev server (vite...) + wire HFD_DEV_URL
  hfd new    <name> [--template react|vue] [--core path]
                       Scaffold a fresh HFD project (default: react + vite)
  hfd doctor [dir]     Check the build environment (Go/node/GTK/WebKitGTK/tray)
  hfd package [dir]    Build + assemble an installable .deb (usr/local/bin + .desktop)
  hfd help             Show this help

Configure via 'hfd.json' in the project dir:
  {
    "app":    { "dir": "." },
    "frontend": { "dir": "frontend", "dev": "npm run dev",
                  "build": "npm run build", "port": 5173 },
    "bindingsDir": "frontend/src/hfd",
    "goBuild": "go build -o build/bin/app ."
  }
`;

export async function main(argv) {
	if (argv[0] === "new") {
		const rest = argv.slice(1);
		const name = rest.find((a) => !a.startsWith("-"));
		const flag = (n, dflt) => {
			const i = rest.indexOf(n);
			return i >= 0 && rest[i + 1] ? rest[i + 1] : dflt;
		};
		const template = flag("--template", "react");
		const corePath = flag("--core", "../../core");
		if (!name) {
			console.error("hfd new needs <name> — usage: hfd new myapp --template react");
			process.exit(1);
		}
		await runNew(name, { template, corePath });
		return;
	}

	const [cmd, dir = "."] = argv;

	switch (cmd) {
		case "bindings":
			await runBindings(dir);
			break;
		case "build":
			await runBuild(dir);
			break;
		case "dev":
			await runDev(dir);
			break;
		case "doctor":
			process.exitCode = runDoctor(dir);
			break;
		case "package":
			await runPackage(dir);
			break;
		case "help":
		case "--help":
		case "-h":
		case undefined:
			console.log(HELP.trim());
			break;
		default:
			throw new Error(`unknown command: ${cmd} (try 'hfd help')`);
	}
}
