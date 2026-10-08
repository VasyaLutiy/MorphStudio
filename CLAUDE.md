# MorphStudio

A go project built by **Morph cards**: the code under `src/` (or the language's code folders) and its tests
are written by Morph's executors from the record, never by hand and never by an agent. The cycle of one phase:
the record `contour.yaml` + `morph-map.json` → `morph plan --spec contour.yaml --map morph-map.json --component <C> --judge
--checks decks/<phase>/checks.json --out decks/<phase>/deck.json` → `morph deck check` → the gate → `morph run --processor
<processor>` (order and flags: `docs/AUTONOMY.md`, "The cycle of one phase"). One code card and one judge card per
Function. The plan by epics and phases: `PLAN.md`; measurements: `docs/MEASURE.md`; gaps decided:
`docs/DECISIONS.md`.

- The `morph` binary: `morph` on PATH, or `node <morph package>/dist/cli.js`. A run uses a COPY of the binary outside the
  repository, so a card that rebuilds cannot replace the running binary.
- Processor keys live outside the repository (`MORPH_PROCESSOR_<P>_<KEY>` in the environment of the run, from a file only
  the operator writes). Never print them.
- The autonomous mode: `docs/AUTONOMY.md`. A phase's spec: `docs/TASK_TEMPLATE.md` → `docs/TASK_<phase>_<component>.md`.
- Written by hand (the agent) are only data: the record, the map, the phase's TASK, fixtures under `tests/fixtures/`,
  `decks/<phase>/checks.json`, probes `decks/<phase>/parts/*`, the lock file after the scaffold, the API digests
  `docs/deps/*.md` of the declared dependencies and what the scaffold installs for them (`vendor/`, the lock file, the
  pinned requirements; `PLAN.md` "What is known" (the SDK) and "Zero Contour" (Dependency)). A hand edit of code breaks the experiment.
- Paid runs: only on the operator's word or by the gate of `docs/AUTONOMY.md`; the cap per phase is in its section
  "Money".
- Before every run: `morph plan` exit 0, `morph deck check` with 0 errors, every acceptance red per example on stubs in a
  scratch worktree.
- Run branches `morph/<run-id>`; `main` moves only by fast-forward. Force-push is forbidden.
- Build artefacts (`.morph/runs/<id>/`, `decks/<phase>/deck.json`) are written by Morph.
