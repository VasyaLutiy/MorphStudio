// First-difference locator of a Go module: reads a `go test` log and, for every failed comparison printed by the
// module's internal/testhelp Equal ("got:  <x>" on one line, "want: <y>" on the next, indented under the failing test's
// line), prints where the two first differ, with 40 chars of context on each side. `morph plan --checks` inlines this
// file into every acceptance.
//   node firstdiff.mjs <go test log>
import fs from "node:fs";

const lines = fs.readFileSync(process.argv[2], "utf8").split("\n");
const pairs = [];
for (let i = 0; i + 1 < lines.length; i++) {
  const got = /^\s+got:\s+(.*)$/.exec(lines[i]);
  const want = /^\s+want:\s+(.*)$/.exec(lines[i + 1]);
  if (got && want) {
    pairs.push([want[1], got[1]]);
    i++;
  }
}
let shown = 0;
for (const [e, r] of pairs) {
  if (e === r || shown >= 6) continue;
  let k = 0;
  while (k < e.length && k < r.length && e[k] === r[k]) k++;
  const at = (s) => JSON.stringify((k > 40 ? "..." : "") + s.slice(Math.max(0, k - 40), k + 40));
  console.log(`first difference at char ${k} of ${e.length}/${r.length}: expected ${at(e)} received ${at(r)}`);
  shown++;
}
