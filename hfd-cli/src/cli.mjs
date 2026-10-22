import { runBindings } from "./cmd-bindings.mjs";
import { runBuild } from "./cmd-build.mjs";
import { runDev } from "./cmd-dev.mjs";
import { runNew } from "./cmd-new.mjs";

const HELP = `
hfd — CLI của Hyggshi Framework Desktop

Lệnh:
  hfd bindings [dir]   Sinh bindings TS type-safe từ Go (BindTyped),
                       đọc schema bằng 'HFD_CODEGEN=bindings go run .'
  hfd build [dir]      Build frontend rồi build binary Go → build/bin/
  hfd dev    [dir]     Chạy dev server (vite...) + dùng env HFD_DEV_URL
  hfd new    <tên> [--template react|vue] [--core path]
                       Scaffold project HFD mới (mặc định: react + vite)
  hfd help             Hiển thị help này

Cấu hình qua 'hfd.json' ở project dir:
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
			console.error("hfd new cần <tên> — dùng: hfd new myapp --template react");
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
		case "help":
		case "--help":
		case "-h":
		case undefined:
			console.log(HELP.trim());
			break;
		default:
			throw new Error(`lệnh không rõ: ${cmd} (gõ 'hfd help')`);
	}
}
