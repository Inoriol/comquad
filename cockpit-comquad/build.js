#!/usr/bin/env node

import esbuild from "esbuild";
import { sassPlugin } from "esbuild-sass-plugin";
import process from "process";
import fs from "fs";
import path from "path";

const production = process.env.NODE_ENV === "production";
const watchMode = process.argv.includes("-w") || process.argv.includes("--watch");

const outdir = "dist";

if (!fs.existsSync(outdir)) {
    fs.mkdirSync(outdir, { recursive: true });
}

const context = await esbuild.context({
    entryPoints: ["src/index.tsx"],
    bundle: true,
    outdir,
    format: "esm",
    platform: "browser",
    target: "es2020",
    loader: {
        ".tsx": "tsx",
        ".ts": "ts",
        ".jsx": "jsx",
        ".js": "js",
        ".png": "file",
        ".svg": "file",
        ".woff": "file",
        ".woff2": "file",
        ".eot": "file",
        ".ttf": "file",
    },
    plugins: [sassPlugin()],
    define: {
        "process.env.NODE_ENV": JSON.stringify(production ? "production" : "development"),
    },
    minify: production,
    sourcemap: !production,
    external: ["cockpit", "cockpit/*"],
});

if (watchMode) {
    await context.watch();
    console.log("Watching for changes...");
} else {
    await context.rebuild();
    await context.dispose();
    
    // Copy static files
    fs.copyFileSync("src/manifest.json", path.join(outdir, "manifest.json"));
    fs.copyFileSync("src/index.html", path.join(outdir, "index.html"));
    
    console.log("Build complete");
}
