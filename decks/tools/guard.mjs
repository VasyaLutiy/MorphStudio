// Guard of a Go module: the layer rules and the Guardrail rules of the record, and the test-file rules. `morph plan
// --checks` inlines this file into every acceptance. Imports and calls are read by the go tool (a go/parser scan of
// each given code file), never by grep: a word in a comment is not an import. Node's standard library only. Every
// rule is checked per FILE, never per package: a red names the importing file and line, so a package-mate's import
// is never this card's red (P31b, issue #42).
// morph-guard-contract: 3
//   node guard.mjs src <file,file,...> <mods> <layers> <rules>  the packages of these Go files, all four arguments
//                                                         written by `morph plan --checks` from the record:
//     mods   = the module paths the card's Component declares (`uses`): a path or a path + "/...", ","-joined ('' none);
//              a file may import the standard library (narrowed only by the rules below) and these, nothing else
//              outside the module;
//     layers = JSON {"<package dir>": ["<dir it may import>", ...]} from the `uses: <Component>` steps (contract 2): a
//              file of a package dir it lacks may import no other package of the module;
//     rules  = JSON {"<package dir>": {"<rule id>": "<the Guardrail and where the record attaches it>"}} from the
//              record's canonical Guardrail names (contract 3).
//              A missing layers or rules argument is red, never "everything allowed" (issue #46): re-cut with morph plan.
//   node guard.mjs tests <file> <min> <max> [lits.json]   one Go test file; lits = strings it must hold; it may import
//                                                         the module's packages and go.mod's direct requirements
// Rule ids (each checked in the given code files, never in a _test.go file; a red names the file and line):
//   no-shell     no import of os/exec
//   no-clock     no time.Now, time.Since, time.Until; no import of math/rand, math/rand/v2, crypto/rand
//   no-network   no import of net, net/http, net/rpc, net/smtp or a package under net/http/, net/rpc/
//   no-env       no os.Getenv, os.LookupEnv, os.Environ, os.Setenv, os.Unsetenv, os.Clearenv, os.ExpandEnv
//   stdlib-only  no import outside the standard library and the module, the declared modules included
//   pure         all of the above but stdlib-only, and no import of os, time, syscall, unsafe
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const bad = [];
const PURE_FORBIDDEN = ["os", "os/exec", "net", "net/http", "time", "math/rand", "math/rand/v2", "syscall", "unsafe"];
const RULE_ORDER = ["no-shell", "no-clock", "no-network", "no-env", "stdlib-only", "pure"];
const CLOCK_IMPORTS = ["math/rand", "math/rand/v2", "crypto/rand"];
const NET_IMPORTS = ["net", "net/http", "net/rpc", "net/smtp"];
const CLOCK_USES = ["time.Now", "time.Since", "time.Until"];
const ENV_USES = ["os.Getenv", "os.LookupEnv", "os.Environ", "os.Setenv", "os.Unsetenv", "os.Clearenv", "os.ExpandEnv"];
const TEST_CALL = /^func Test(?:[A-Z0-9_]\w*)?\s*\(/gm;

function modulePath() {
  const text = fs.existsSync("go.mod") ? fs.readFileSync("go.mod", "utf8") : "";
  const m = /^module\s+(\S+)/m.exec(text);
  if (!m) bad.push("guard: go.mod has no module line");
  return m ? m[1] : "";
}
function isStd(imp) {
  return !imp.split("/")[0].includes(".");
}
function checkSrc(files, allowed, layers, rules) {
  const mod = modulePath();
  const code = [];
  for (const f of files) {
    const rel = f.split(path.sep).join("/");
    if (!rel.endsWith("_test.go")) code.push([f, rel]);
  }
  if (code.length === 0) return;
  const scanned = scanFiles(code.map(([f]) => f));
  for (const [f, rel] of code) {
    const r = scanned[f];
    if (!r) { bad.push(`guard: ${rel}: not scanned`); continue; }
    if (r.error) { bad.push(`guard: ${rel}: does not parse: ${r.error}`); continue; }
    const dir = path.posix.dirname(rel);
    const active = rules[dir] ?? {};
    const first = (ids) => RULE_ORDER.find((id) => ids.includes(id) && id in active);
    for (const h of r.imports) {
      const imp = h.name;
      const id = first(importRules(imp, mod));
      if (id) { bad.push(`guard: ${rel}:${h.line}: imports ${imp}: ${active[id]}`); continue; }
      if (imp === mod || imp.startsWith(mod + "/")) {
        const target = imp === mod ? "." : imp.slice(mod.length + 1);
        if (target === dir) continue;
        const may = layers[dir] ?? [];
        if (!may.includes(target))
          bad.push(`guard: ${rel}:${h.line}: imports ${imp}: the record has no 'uses' from ${dir} to ${target} (allowed: ${may.join(", ") || "none"})`);
        continue;
      }
      if (!isStd(imp) && !allowed.some((m) => imp === m || imp.startsWith(m + "/")))
        bad.push(`guard: ${rel}:${h.line}: imports ${imp}: not a dependency the record gives this Component (allowed: the standard library${allowed.length ? ", " + allowed.join(", ") : ""})`);
    }
    for (const h of r.uses) {
      const id = first(useRules(h.name));
      if (id) bad.push(`guard: ${rel}:${h.line}: uses ${h.name}: ${active[id]}`);
    }
  }
}
// The go/parser scan of the code files: per file its imports and every selector <imported package>.<Name>, with lines.
const SCAN = `package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type hit struct {
	Name string \`json:"name"\`
	Line int    \`json:"line"\`
}
type file struct {
	Error   string \`json:"error,omitempty"\`
	Imports []hit  \`json:"imports"\`
	Uses    []hit  \`json:"uses"\`
}

var version = regexp.MustCompile("^v[0-9]+$")

func main() {
	out := map[string]file{}
	fset := token.NewFileSet()
	for _, p := range os.Args[1:] {
		r := file{Imports: []hit{}, Uses: []hit{}}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			r.Error = err.Error()
			out[p] = r
			continue
		}
		names := map[string]string{}
		for _, s := range f.Imports {
			ip, _ := strconv.Unquote(s.Path.Value)
			r.Imports = append(r.Imports, hit{ip, fset.Position(s.Pos()).Line})
			parts := strings.Split(ip, "/")
			name := parts[len(parts)-1]
			if version.MatchString(name) && len(parts) > 1 {
				name = parts[len(parts)-2]
			}
			if s.Name != nil {
				name = s.Name.Name
			}
			names[name] = ip
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					if ip, ok := names[id.Name]; ok {
						r.Uses = append(r.Uses, hit{ip + "." + sel.Sel.Name, fset.Position(sel.Pos()).Line})
					}
				}
			}
			return true
		})
		out[p] = r
	}
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
`;
function scanFiles(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "morph-guard-"));
  try {
    fs.writeFileSync(path.join(dir, "go.mod"), "module morphguardscan\n\ngo 1.21\n");
    fs.writeFileSync(path.join(dir, "main.go"), SCAN);
    const out = execFileSync("go", ["run", ".", ...files.map((f) => path.resolve(f))], {
      cwd: dir, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, GOFLAGS: "-mod=mod", GOWORK: "off" },
    });
    const doc = JSON.parse(out);
    const byFile = {};
    for (const f of files) byFile[f] = doc[path.resolve(f)];
    return byFile;
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}
function importRules(imp, mod) {
  const out = [];
  if (imp === "os/exec") out.push("no-shell");
  if (CLOCK_IMPORTS.includes(imp)) out.push("no-clock");
  const net = NET_IMPORTS.includes(imp) || imp.startsWith("net/http/") || imp.startsWith("net/rpc/");
  if (net) out.push("no-network");
  if (!isStd(imp) && imp !== mod && !imp.startsWith(mod + "/")) out.push("stdlib-only");
  if (imp === "os/exec" || CLOCK_IMPORTS.includes(imp) || net || PURE_FORBIDDEN.includes(imp)) out.push("pure");
  return out;
}
function useRules(sym) {
  const out = [];
  if (CLOCK_USES.includes(sym)) out.push("no-clock", "pure");
  if (ENV_USES.includes(sym)) out.push("no-env", "pure");
  return out;
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
if (mode === "src" && c === undefined) bad.push("guard: no layer edges given — re-cut with morph plan");
else if (mode === "src" && d === undefined) bad.push("guard: no Guardrail rules given — re-cut with morph plan");
else if (mode === "src") checkSrc((a ?? "").split(",").filter(Boolean), (b ?? "").split(",").filter(Boolean), JSON.parse(c), JSON.parse(d));
else if (mode === "tests") checkTest(a, Number(b), Number(c), d ? JSON.parse(fs.readFileSync(d, "utf8")) : []);
else bad.push(`guard: unknown mode ${mode}`);
for (const x of bad) console.log(x);
process.exit(bad.length ? 1 : 0);
