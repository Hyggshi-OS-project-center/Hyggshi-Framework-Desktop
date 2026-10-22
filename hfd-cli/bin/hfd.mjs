#!/usr/bin/env node
// Published entry — only imports the compiled output (dist/), so running the
// CLI needs zero npm install. Rebuild dist with: (cd hfd-cli && npm run build)
import { main } from "../dist/cli.js";

main(process.argv.slice(2)).catch((err) => {
	console.error("hfd:", err.message);
	process.exit(1);
});
