// The gate's stub check of one card: reads the log of the card's acceptance run on a one-test stub and
// says whether it is red where the stub must make it red, and only there. Node's standard library only.
//   node decks/tools/stubcheck.mjs <log> <stage> <target,target,...>
// <stage>: the stage the stub must stop at — `probe` for a code card, `guard` for a judge before its file. The log is the
// acceptance's whole output; its stages are the lines "== <stage>[ <file>]" the acceptance echoes before each step (a
// held lint verdict, "== vet or gofmt failed (see above)…" or "== eslint failed (see above)…", is the stage `verdict`).
// Fails (exit 1, one line each) when
//   - the last stage the log reached is not <stage> (the red came earlier or later, or the run was green);
//   - a line of a compile stage (build, vet, tsc) names a source file outside the targets, e.g. a Go vet line
//     "pkg/mount.go:49:2: undefined: SessionServer" under "== vet" while the guard reddens later: an accepted file
//     broken by the card's overlay is a defect of the deck, never an expected red.
// Exit 0 prints "stubcheck: red at <stage>; build, vet and tsc lines name only the targets".
import fs from "node:fs";

const [logPath, stage, targetArg] = process.argv.slice(2);
if (!logPath || !stage) {
  console.log("usage: node stubcheck.mjs <log> <stage> <target,target,...>");
  process.exit(2);
}
const targets = (targetArg ?? "").split(",").filter(Boolean).map((t) => t.replace(/^\.\//, ""));
const COMPILE = new Set(["build", "vet", "tsc"]);
const SOURCE = /((?:\.{0,2}\/)?[\w.@+-]+(?:\/[\w.@+-]+)*\.(?:go|ts|tsx|mts|js|mjs|py))(?::\d+|\(\d+,\d+\))/g;

const bad = [];
let current = null;
let last = null;
for (const line of fs.readFileSync(logPath, "utf8").split("\n")) {
  const header = /^== (\S+)/.exec(line);
  if (header) {
    current = / failed \(see above\)/.test(line) ? "verdict" : header[1];
    last = current;
    continue;
  }
  if (current === null || !COMPILE.has(current)) continue;
  for (const m of line.matchAll(SOURCE)) {
    const file = m[1].replace(/^\.\//, "");
    if (!targets.includes(file)) {
      bad.push(`stubcheck: ${current} names ${file}, outside the targets: ${line.trim()}`);
      break;
    }
  }
}
if (last !== stage) bad.push(`stubcheck: the log stops at stage ${last ?? "(none)"}, expected ${stage}`);
for (const b of bad) console.log(b);
if (bad.length === 0) console.log(`stubcheck: red at ${stage}; build, vet and tsc lines name only the targets`);
process.exit(bad.length ? 1 : 0);
