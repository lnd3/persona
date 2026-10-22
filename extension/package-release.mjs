// Packages a distributable zip of this extension -- exactly what a
// user needs for "Load unpacked" (manifest.json, dist/, the two popup
// HTML files), nothing more. Interim distribution method per A012's
// own Decisions (Chrome Web Store / Firefox-signed self-distribution
// are both still unbuilt) -- see plan/actions/A012's own Log for why
// this exists and what it replaces (a manual build+zip sequence that
// had already drifted out of sync with the source once).
//
// Usage: npm run package
// Output: release/persona-signer-v<version>.zip

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, cpSync, statSync, createWriteStream } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { ZipArchive } from "archiver";

const ROOT = import.meta.dirname;

function run(cmd, args) {
  execFileSync(cmd, args, { cwd: ROOT, stdio: "inherit" });
}

// Always rebuild first -- a packaged release must never ship a dist/
// that's gone stale relative to src/. (This exact mistake -- testing
// against an old dist/ after editing src/ -- happened manually more
// than once while building this extension; the packager shouldn't
// have the same failure mode.)
console.log("==> Building...");
run("node", ["build.mjs"]);

const manifest = JSON.parse(readFileSync(path.join(ROOT, "manifest.json"), "utf8"));
const version = manifest.version;
if (!version) {
  throw new Error("manifest.json has no version field");
}

const staging = mkdtempSync(path.join(tmpdir(), "persona-signer-release-"));
console.log(`==> Staging in ${staging}...`);

// Exactly what manifest.json and the two popup pages reference --
// never the whole extension/ tree (no node_modules, no .ts source,
// no build tooling).
cpSync(path.join(ROOT, "manifest.json"), path.join(staging, "manifest.json"));
cpSync(path.join(ROOT, "dist"), path.join(staging, "dist"), { recursive: true });
mkdirSync(path.join(staging, "src", "popup"), { recursive: true });
mkdirSync(path.join(staging, "src", "approve"), { recursive: true });
cpSync(path.join(ROOT, "src", "popup", "popup.html"), path.join(staging, "src", "popup", "popup.html"));
cpSync(path.join(ROOT, "src", "approve", "approve.html"), path.join(staging, "src", "approve", "approve.html"));
if (existsSync(path.join(ROOT, "README.md"))) {
  cpSync(path.join(ROOT, "README.md"), path.join(staging, "README.md"));
}

const releaseDir = path.join(ROOT, "release");
mkdirSync(releaseDir, { recursive: true });
const zipPath = path.join(releaseDir, `persona-signer-v${version}.zip`);

console.log(`==> Zipping to ${zipPath}...`);
await new Promise((resolve, reject) => {
  const output = createWriteStream(zipPath);
  const archive = new ZipArchive({ zlib: { level: 9 } });
  output.on("close", resolve);
  archive.on("error", reject);
  archive.pipe(output);
  archive.directory(staging, false); // false: files land at the zip root, not under a wrapper folder
  void archive.finalize();
});

rmSync(staging, { recursive: true, force: true });

const sizeKb = (statSync(zipPath).size / 1024).toFixed(1);
console.log(`\n==> Done: ${zipPath} (${sizeKb} KB)`);
console.log(`    Load unpacked: extract the zip, then "Load unpacked" on the extracted folder.`);
