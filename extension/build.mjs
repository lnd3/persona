import * as esbuild from "esbuild";
import { execSync } from "node:child_process";

const watch = process.argv.includes("--watch");

// Same shape as deploy/deploy.sh's own ${PERSONA_BUILD_INFO} ("build
// <short-rev> · <UTC time>") so a build id means the same thing
// whether it's read off the public site or this extension's own
// popup. Falls back gracefully (e.g. a tarball with no .git) rather
// than failing the whole build over a cosmetic string.
function buildInfo() {
  let rev = "unknown";
  try {
    rev = execSync("git rev-parse --short HEAD", { cwd: import.meta.dirname }).toString().trim();
  } catch {
    // no .git available (e.g. built from a source tarball) -- "unknown" stands.
  }
  const time = new Date().toISOString().replace(/\.\d+Z$/, "Z");
  return `build ${rev} · ${time}`;
}

const options = {
  entryPoints: [
    "src/background.ts",
    "src/content.ts",
    "src/inject.ts",
    "src/popup/popup.ts",
    "src/approve/approve.ts",
  ],
  bundle: true,
  format: "esm",
  target: "chrome120",
  outdir: "dist",
  sourcemap: true,
  logLevel: "info",
  define: {
    __BUILD_INFO__: JSON.stringify(buildInfo()),
  },
};

if (watch) {
  const ctx = await esbuild.context(options);
  await ctx.watch();
  console.log("watching for changes...");
} else {
  await esbuild.build(options);
}
