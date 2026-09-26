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
    
    // Generate po.js language loader
    const poJsContent = `// Auto-detect language and load appropriate translation
(function() {
    const lang = navigator.language || navigator.userLanguage || 'en';
    const shortLang = lang.split('-')[0];
    const fullLang = shortLang + '_' + (lang.split('-')[1] || '').toUpperCase();
    
    // Try full language code first, then short code, then English
    const langsToTry = [fullLang, shortLang, 'en'];
    
    for (const tryLang of langsToTry) {
        const script = document.createElement('script');
        script.src = \`po.\${tryLang}.js\`;
        script.onerror = function() {
            // If this language file doesn't exist, try the next one
            console.log(\`Translation file po.\${tryLang}.js not found, trying next...\`);
        };
        script.onload = function() {
            console.log(\`Loaded translations for \${tryLang}\`);
        };
        document.head.appendChild(script);
        return; // Only load one language file
    }
})();`;
    
    fs.writeFileSync(path.join(outdir, "po.js"), poJsContent);
    
    // Compile po files to JS
    const poDir = "po";
    if (fs.existsSync(poDir)) {
        const linguasPath = path.join(poDir, "LINGUAS");
        let linguas = [];
        if (fs.existsSync(linguasPath)) {
            const linguasContent = fs.readFileSync(linguasPath, "utf-8");
            linguas = linguasContent.split("\n")
                .map(line => line.trim())
                .filter(line => line && !line.startsWith("#"));
        }
        
        for (const lang of linguas) {
            const poFile = path.join(poDir, `${lang}.po`);
            if (fs.existsSync(poFile)) {
                const poContent = fs.readFileSync(poFile, "utf-8");
                const jsContent = parsePoToJs(poContent, lang);
                fs.writeFileSync(
                    path.join(outdir, `po.${lang}.js`),
                    jsContent
                );
            }
        }
    }
    
    console.log("Build complete");
}

function parsePoToJs(content, lang) {
    const RTL_LANGS = new Set(["ar", "fa", "he", "ur"]);
    const direction = RTL_LANGS.has(lang) ? "rtl" : "ltr";
    
    const translations = {
        "": {
            "language": lang,
            "plural-forms": "",
            "language-direction": direction
        }
    };
    const entries = content.split(/\n\n+/);

    const headerMatch = content.match(/^msgid\s+""\s*\n(?:msgstr\s+""\s*\n)?((?:"[^"]*"\s*\n)+)/m);
    if (headerMatch) {
        const headerLines = headerMatch[1];
        for (const line of headerLines.split('\n')) {
            const lineMatch = line.match(/^"(.*)"$/);
            if (lineMatch) {
                const lineContent = lineMatch[1].replace(/\\n/g, '\n');
                const langMatch = lineContent.match(/^Language:\s*(\S+)/);
                if (langMatch) {
                    translations[""].language = langMatch[1];
                }
                const pluralMatch = lineContent.match(/^Plural-Forms:\s*(.*)/);
                if (pluralMatch) {
                    const pluralForms = pluralMatch[1];
                    const expr = pluralForms.replace(/nplurals=[1-9];\s*plural=([^;]*);?$/, '(n) => $1');
                    translations[""]["plural-forms"] = expr;
                }
            }
        }
    }

    for (const entry of entries) {
        const msgidMatch = entry.match(/^msgid\s+"(.*)"/m);
        const msgstrMatches = [...entry.matchAll(/^msgstr(?:\[(\d+)\])?\s+"(.*)"/gm)];

        if (msgidMatch && msgstrMatches.length > 0) {
            const msgid = msgidMatch[1];
            if (!msgid) continue;

            const msgstrArray = [null];
            for (const match of msgstrMatches) {
                msgstrArray.push(match[2]);
            }
            
            if (msgstrArray.length > 1) {
                translations[msgid] = msgstrArray;
            }
        }
    }

    const chunks = ['{\n'];
    chunks.push(' "": {\n');
    chunks.push(`  "plural-forms": ${translations[""]["plural-forms"]},\n`);
    chunks.push(`  "language": "${translations[""].language}",\n`);
    chunks.push(`  "language-direction": "${direction}"\n`);
    chunks.push(' }');

    for (const [msgid, msgstrArray] of Object.entries(translations)) {
        if (msgid === "") continue;
        
        const key = JSON.stringify(msgid);
        chunks.push(`,\n ${key}: [\n  null`);
        for (let i = 1; i < msgstrArray.length; i++) {
            chunks.push(`,\n  ${JSON.stringify(msgstrArray[i])}`);
        }
        chunks.push('\n ]');
    }

    chunks.push('\n}');

    return `cockpit.locale(${chunks.join('')});\n`;
}
