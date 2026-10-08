# Measurements by phase

One row per phase (and per fix, smoke or debt as its own row). Money in dollars of the provider's bill; the orchestrator
column as "<model> <tokens>/<tool calls>/<minutes>". The running total of the autonomous stretch is kept under the table.

| phase | builder | cards planned / accepted / burned | $ executor / $ orchestrator | minutes | tsc-first-red | neighbour-red | judge defects | judge stubs | lines by hand | hazards | max slice, bytes | failure class | runs |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| P1 stream, runner, queue | ds (deepseek-v4.1-flash), binary copy of MorphV2 bc311aa | 8 / 8 / 0 | 0.0495 / Opus 5.5 orchestrator agent 196k tok/69 calls/12 min (prep) | run 5.3, prep 12 | 0 | 0 | 0 (read: 2 defects vs §2.2, recorded) | 0 | 0 | 0 | 84 876 | — (1 retry: phase-queue build, first attempt red) | 20261008-171107 |

Running total: $0.0495 of $35 (P1 $0.0495).
