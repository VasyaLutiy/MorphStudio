// Guard of a Go module: the layer rules of decks/tools/layers.json and the test-file rules. `morph plan --checks`
// inlines this file into every acceptance. Imports are read by the go tool (`go list`), never by grep: a word in a
// comment is not an import. Node's standard library only.
//   node guard.mjs src <file,file,...> [mod,mod,...]      the packages of these Go files; the module paths the card's
//                                                         Component declares (`uses`): a path or a path + "/..."
//   node guard.mjs tests <file> <min> <max> [lits.json]   one Go test file; lits = strings it must hold; it may import
//                                                         the module's packages and go.mod's direct requirements
// decks/tools/layers.json: {"layers": {"<dir>": ["<dir it may import>", ...]}, "pure": ["<dir>", ...]};
// "layers" null or the file absent → every package of the module may import every other; no package is pure.
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const bad = [];
const PURE_FORBIDDEN = ["os", "os/exec", "net", "net/http", "time", "math/rand", "math/rand/v2", "syscall", "unsafe"];
const TEST_CALL = /^func Test(?:[A-Z0-9_]\w*)?\s*\(/gm;

function modulePath() {
  const text = fs.existsSync("go.mod") ? fs.readFileSync("go.mod", "utf8") : "";
  const m = /^module\s+(\S+)/m.exec(text);
  if (!m) bad.push("guard: go.mod has no module line");
  return m ? m[1] : "";
}
function readLayers() {
  const file = "decks/tools/layers.json";
  if (!fs.existsSync(file)) return { layers: null, pure: [] };
  const doc = JSON.parse(fs.readFileSync(file, "utf8"));
  return { layers: doc.layers ?? null, pure: doc.pure ?? [] };
}
function isStd(imp) {
  return !imp.split("/")[0].includes(".");
}
function goImports(dir) {
  const out = execFileSync("go", ["list", "-e", "-f", '{{join .Imports "\\n"}}', "./" + dir], {
    encoding: "utf8", stdio: ["ignore", "pipe", "pipe"],
  });
  return out.split("\n").filter(Boolean);
}
function checkSrc(files, allowed) {
  const mod = modulePath();
  const { layers, pure } = readLayers();
  const dirs = [];
  for (const f of files) {
    const d = path.posix.dirname(f.split(path.sep).join("/"));
    if (!dirs.includes(d)) dirs.push(d);
  }
  for (const dir of dirs) {
    if (layers !== null && !(dir in layers)) {
      bad.push(`guard: ${dir} is not a package of decks/tools/layers.json (have: ${Object.keys(layers).join(", ")})`);
      continue;
    }
    for (const imp of goImports(dir)) {
      if (imp === mod || imp.startsWith(mod + "/")) {
        const target = imp === mod ? "." : imp.slice(mod.length + 1);
        if (target === dir) continue;
        if (layers !== null && !layers[dir].includes(target))
          bad.push(`guard: package ${dir} imports ${imp} (allowed: ${layers[dir].join(", ") || "none"})`);
        continue;
      }
      if (!isStd(imp)) {
        if (allowed.some((m) => imp === m || imp.startsWith(m + "/"))) continue;
        bad.push(`guard: package ${dir} imports ${imp} (the standard library ${allowed.length ? "and " + allowed.join(", ") + " " : ""}only)`);
        continue;
      }
      if (pure.includes(dir) && PURE_FORBIDDEN.includes(imp))
        bad.push(`guard: package ${dir} imports ${imp} in a pure package (no ${PURE_FORBIDDEN.join(", ")})`);
    }
  }
}
function fileImports(text) {
  const out = [];
  for (const m of text.matchAll(/^import\s+(?:[\w.]+\s+)?"([^"]+)"/gm)) out.push(m[1]);
  for (const m of text.matchAll(/^import\s*\(([\s\S]*?)^\)/gm))
    for (const line of m[1].split("\n")) {
      const s = /^\s*(?:[\w.]+\s+)?"([^"]+)"/.exec(line.replace(/\/\/.*$/, ""));
      if (s) out.push(s[1]);
    }
  return out;
}
// go.mod's direct requirements: the module paths of its `require` lines and blocks, without the
// lines marked `// indirect`. A test file may import them (a judge needs a declared module's test doubles, e.g. an
// in-memory transport); the scaffold vendored them, so the test builds offline.
function directRequires() {
  const text = fs.existsSync("go.mod") ? fs.readFileSync("go.mod", "utf8") : "";
  const out = [];
  const take = (line) => {
    if (/\/\/\s*indirect\b/.test(line)) return;
    const m = /^\s*([^\s/][^\s]*)\s+v\S+/.exec(line.replace(/\/\/.*$/, ""));
    if (m && !out.includes(m[1])) out.push(m[1]);
  };
  for (const m of text.matchAll(/^require\s*\(([\s\S]*?)^\)/gm)) for (const line of m[1].split("\n")) take(line);
  for (const m of text.matchAll(/^require\s+([^(\s].*)$/gm)) take(m[1]);
  return out;
}
function checkTest(file, min, max, lits) {
  if (!fs.existsSync(file)) { bad.push(`guard: ${file} missing`); return; }
  const text = fs.readFileSync(file, "utf8");
  const mod = modulePath();
  const requires = directRequires();
  for (const imp of fileImports(text))
    if (!isStd(imp) && !(imp === mod + "/internal/testhelp" || imp.startsWith(mod + "/")) &&
        !requires.some((r) => imp === r || imp.startsWith(r + "/")))
      bad.push(`guard: ${file} imports ${imp} (the standard library${requires.length ? ", " + mod + "/internal/testhelp, " + requires.join(", ") : " and " + mod + "/internal/testhelp"} only)`);
  const code = text.split("\n").map((l) => l.replace(/\/\/.*$/, "")).join("\n");
  for (const m of code.matchAll(/\.(Skip|Skipf|SkipNow)\(|\btesting\.Short\(/g))
    bad.push(`guard: ${file} calls ${m[1] ?? "testing.Short"} (a test runs or fails, it is never skipped)`);
  const count = (text.match(TEST_CALL) ?? []).length;
  if (count < min || count > max) bad.push(`guard: ${file} has ${count} func Test, expected ${min}..${max}`);
  for (const lit of lits) if (!text.includes(lit)) bad.push(`guard: ${file} does not mention the example literal ${JSON.stringify(lit)}`);
}

const [mode, a, b, c, d] = process.argv.slice(2);
if (mode === "src") checkSrc((a ?? "").split(",").filter(Boolean), (b ?? "").split(",").filter(Boolean));
else if (mode === "tests") checkTest(a, Number(b), Number(c), d ? JSON.parse(fs.readFileSync(d, "utf8")) : []);
else bad.push(`guard: unknown mode ${mode}`);
for (const x of bad) console.log(x);
process.exit(bad.length ? 1 : 0);
