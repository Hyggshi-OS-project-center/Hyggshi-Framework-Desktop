import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// Build output ./dist — go:embed all:frontend/dist nhúng vào binary.
export default defineConfig({
	plugins: [react()],
	build: { outDir: "dist" },
});
