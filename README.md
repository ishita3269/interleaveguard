# interleaveguard tasks
INTERLEAVEGUARD - PROJECT 4 README
==================================
LLM-assisted compositional verification and concurrency vulnerability
detection in Go.
 
Team:      Ishita and Vedhant
Deadlines: 14 Nov 2026  - 10-minute progress update to the TAs
           27-29 Nov    - final in-person presentations
 
 
1. WHAT THE PROJECT IS
----------------------
Some concurrency bugs live in code that passes Go's race detector, because
every memory access is locked. The bug is that a bigger check-then-update
step is not atomic, so two goroutines can both pass a check and both act.
 
Example (task1/account.go): Withdraw checks the balance under a lock, releases
the lock, then subtracts under a second lock. Two Withdraw(100) calls on a
balance of 100 can both succeed and leave -100.
 
We use an LLM to infer what each function should guarantee when other threads
run at the same time (rely-guarantee-style contracts), pick the few dangerous
schedules, turn them into executable Go tests, and report the failing test.
Then we measure whether this beats the baselines.
 
Research gap: FM-Agent writes Hoare-style specs from callers but only for
sequential functions. Its Section 11 lists rely-guarantee reasoning as future
work. We test that extension with bounded testing, not a full prover.
 
Research questions:
  RQ1. Do caller-derived specs find more bugs than implementation-derived specs?
  RQ2. Does LLM-guided interleaving selection beat random testing and the
       race detector under the same per-scenario budget?
 
Scope limit: bounded testing only. No claim of proving code correct. Claims
cover only the schedules, operations and properties actually checked.
 
 
2. THE FIVE COMPONENTS (final system)
-------------------------------------
  1. Spec synthesizer          (LLM)        -> rely-guarantee contracts
  2. Shared-state analyzer     (static)     -> shared-state graph
  3. Interleaving explorer     (LLM+rules)  -> ranked risky schedules
  4. Test + counterexample gen (go test)    -> failing Go tests
  5. Debug and repair assistant (LLM)       -> root-cause report
 
Rule: the LLM proposes; only an executed failing test counts as a finding.
 
 
3. REPO LAYOUT (planned)
------------------------
  task1/                      first bug + forced-interleaving test (done)
  benchmark/bNN_name/         one folder per scenario
      code.go
      code_test.go
      truth.md                intended behaviour, bug or "no bug", failing
                              schedule, bug class
  tools/                      spec synthesizer, analyzer, explorer, generator
  results/                    logs, metrics, tables
  report/                     paper draft
 
 
4. COMMANDS
-----------
  go mod init interleaveguard/task1      make a module (once per folder)
  go test -v                             run tests
  go test -race -v                       run with the race detector
  go test -run TestWithdrawForced -v     run one test
  go test -count=20 -run Forced -v       repeat 20 times (reproducibility)
  git add . && git commit -m "msg" && git push
 
Environment: GitHub Codespaces (the Windows laptop blocks the Go linker and
WSL2 cannot start). Both partners use the same repo and codespace setup.
 
 
5. TASKS IN DETAIL
==================
 
TASK 1 - SETUP AND BASELINE                                   [DONE]
------------------------------------------------------------------
Goal: prove the problem is real and reproducible before building anything.
 
What was done:
  - Created task1 module and account.go (Account with mutex, Withdraw with a
    non-atomic check and update, Balance getter).
  - TestWithdrawNormal: two goroutines Withdraw(100) on balance 100. Passes,
    so ordinary testing misses the bug.
  - Added a test-only hook between the check and the update.
  - TestWithdrawForcedInterleaving: a WaitGroup barrier makes both goroutines
    finish the check before either updates. Fails every time:
    "both withdrawals succeeded", "balance went negative: -100".
  - go test -race -v: same results and NO data race warning. First baseline
    data point: race detector finds 0 of 1.
 
Why the barrier works: barrier.Add(2); each goroutine calls Done() then Wait()
in the gap. The first parks; the second brings the counter to 0; both continue.
 
Limits: single hand-written case; hook edits the code under test; one data
point is an illustration, not a statistic.
 
 
TASK 2 - BENCHMARK                                            [NEXT]
------------------------------------------------------------------
Goal: build the dataset every later number is measured on: 15-20 small Go
modules (30-80 lines each), 10-12 with a seeded bug, the rest clean controls.
 
Steps:
  1. Fix the folder layout (section 3).
  2. Write each seeded-bug module around a realistic API (balance, quota,
     job queue, seat map).
  3. Write a forced-schedule test per bug that fails reliably.
  4. Write clean controls. The SAME forced schedule must pass on them.
  5. Write truth.md for every module.
  6. Freeze a split now: modules for developing prompts vs held-out modules
     for final evaluation.
 
Proposed modules (changeable):
  01 Bank Withdraw (done)                    atomicity violation
  02 Transfer A->B and B->A                  deadlock (lock-order inversion)
  03 Seat/ticket reservation                 atomicity (double booking)
  04 Quota check then consume                TOCTOU
  05 Permission check then act               TOCTOU
  06 Counter read/written under diff locks   lost update
  07 Inventory decrement                     lost update / atomicity
  08 Job queue, two workers claim same job   check-then-act
  09 Resource pool allocate                  check-then-act
  10 Shared counter with no lock             data race (race detector should
                                             catch it; keeps comparison fair)
  11 Three locks, inconsistent order         deadlock
  12 Channel + mutex held while sending      deadlock
  C1-C6+ Correct versions: lock held across check and update, ordered locking,
         atomic counter, claim under lock, RWMutex used correctly, seat map in
         one critical section
 
Pitfalls:
  - Strip comments like "BUG" and test-only names (e.g. hook) before anything
    goes to the LLM, or the benchmark is contaminated.
  - Make controls look suspicious (locks, many goroutines) so the false
    positive rate means something.
  - The person who wrote a bug should not be the only judge of it.
 
Suggested split: Ishita writes buggy modules + forced tests; Vedhant writes
matching controls + truth files and the test-runner script.
 
Done when: at least 15 modules, every bug has a failing forced test, every
control passes it, every module has a truth file.
 
 
TASK 3 - SPECIFICATION SYNTHESIZER (LLM)
------------------------------------------------------------------
Goal: for each function, produce a machine-readable rely-guarantee-style
contract from the code and its callers.
 
Steps:
  1. Parse Go code with go/ast: functions, mutex-guarded fields, call sites.
  2. Build a prompt per function: body, callers, shared fields it touches,
     doc comments (hints stripped).
  3. Ask for JSON and validate against a schema.
  4. Run the ABLATION arm: same task with only the function's own code,
     no callers.
  5. Grade at least 10 specs against hand-written intent. Target: >= 70%
     semantically correct.
  6. Log model, prompt, temperature and tokens for every call.
 
Example contract:
  {
    "function": "Account.Withdraw",
    "shared_state": ["balance"],
    "intended_atomic_region": "check balance >= amount AND subtract amount",
    "rely": "other goroutines may call Withdraw/Balance at any time; balance
             changes only under mu",
    "guarantee": "balance >= 0 at all times",
    "postcondition": "returns true => exactly 'amount' was deducted"
  }
 
Why callers matter: callers treat a true result as "funds secured". That
expectation exposes the missing atomicity; the implementation alone looks safe.
 
Risks: hallucinated specs, run-to-run variation. Use low temperature, repeat
runs, schema validation, and never treat a spec as a finding until a test fails.
 
Done when: >= 10 contracts generated and graded, and the implementation-only
ablation exists for the same functions.
 
 
TASK 4 - INTERLEAVING EXPLORER AND TEST GENERATOR (core)
------------------------------------------------------------------
Goal: turn contracts into a short list of dangerous schedules and executable
tests that confirm or reject each one.
 
Steps:
  1. Static analysis: shared variables, guarding mutex, read-modify-write,
     check-then-act shapes, lock acquisition order (a cycle in the lock-order
     graph is a deadlock candidate).
  2. LLM proposal: from the contract and a compact summary, a small ranked
     list of schedules, each an ordered list of (goroutine, step) events.
  3. Deterministic filter: reject schedules that break program order, take a
     lock another goroutine holds, or exceed the bounds (e.g. <= 3 goroutines,
     fixed operation count).
  4. Test generation: emit a Go test that forces the schedule with barriers or
     channels. Run with -race and with -count=20.
  5. Linearizability: for selected stateful APIs, record concurrent histories
     and check them with Porcupine against a simple sequential model.
  6. Classify: reproducible failing test = confirmed finding. Anything else is
     unconfirmed and not counted as a detection.
 
Hard design problem: forcing a schedule WITHOUT hand-editing the code. Options:
  (a) generate an instrumented copy with yield points at lock/unlock
      boundaries,
  (b) replace sync.Mutex with a controllable wrapper in the copy,
  (c) place hooks automatically at points the analyzer flags.
State the choice clearly. Instrumentation can change timing (probe effect);
list it as a threat to validity.
 
Done when: from a repo, the tool outputs a failing, repeatable Go test for
>= 70% of the bugs it correctly detects.
 
 
TASK 5 - EVALUATION, REPORT AND DEMO
------------------------------------------------------------------
Goal: show with numbers whether the method beats the baselines; package it as
a paper-style report and a working demo.
 
Steps:
  1. Run all five methods on the same benchmark with the SAME per-scenario
     budget (time and LLM tokens):
       - InterleaveGuard (caller-derived specs)
       - Ablation: implementation-derived specs
       - go test -race
       - randomized concurrent testing
       - LLM-only code review
  2. Compute the metrics below for each method.
  3. Evaluate on the held-out set only once, after the method is frozen.
  4. Write the report: abstract, introduction, background (Hoare logic,
     FM-Agent, rely-guarantee), approach, benchmark, evaluation, results,
     threats to validity, related work, conclusion.
  5. Prepare the demo: repo in -> spec -> suspicious interleaving -> failing
     test -> report. Rehearse on a fixed example.
  6. Check related work before claiming novelty.
 
Metrics:
  Bug recall            detected seeded reproducible bugs / total
  Precision             confirmed true bugs / everything flagged
  False-positive rate   controls flagged / total controls
  Interleavings explored  schedules executed (to first failure and in total)
  Time to first failure  wall-clock time to the first failing test
  Test reproducibility   fraction of repeated runs where the failing test
                         fails again under the bounded schedules
  LLM token cost         input + output tokens per scenario
 
Targets from the brief:
  - Go goroutines, mutexes, channels; shared-state APIs only
  - 15-20 scenarios; >= 10-12 seeded bugs; plus controls
  - >= 10 specs, >= 70% judged correct
  - detect >= 75% of reproducible seeded bugs; false positives <= 20%
  - auto-generated failing tests for >= 70% of correctly detected bugs
  - compare vs race detector, randomized testing, LLM-only review, plus the
    spec-source ablation
  - end-to-end demo
 
 
6. SUGGESTED SCHEDULE (our proposal; only Nov 14 and Nov 27-29 are fixed)
-------------------------------------------------------------------------
  T1 Setup and baseline            Oct 30 - Nov 2     done
  T2 Benchmark                     Nov 1  - Nov 12
  T3 Spec synthesizer              Nov 6  - Nov 16
  T4 Explorer + test generator     Nov 12 - Nov 23
  T5 Evaluation, report, demo      Nov 18 - Nov 27
 
Minimum to show on Nov 14:
  - Repo shared, Task 1 committed
  - 4-5 benchmark modules with truth files (include one deadlock, one TOCTOU)
  - One LLM-generated contract for Withdraw, caller-derived vs
    implementation-only side by side
  - Next steps and the split of work
 
 
7. KNOWN LIMITATIONS (for the report's threats-to-validity section)
-------------------------------------------------------------------
  - Small, self-made benchmark; bugs we write may suit our own tool
  - Bounded checking, not proof
  - LLM nondeterminism and token cost
  - Prompt contamination from comments or names
  - Instrumentation can change timing (probe effect)
  - Go only, shared-state APIs only
  - Porcupine needs a hand-written sequential model per API
  - Race detector is dynamic; its score depends on how many runs it gets
  - Spec correctness is graded by us (use a second grader on a sample)
  - Novelty not yet verified against related work
 
 
8. GLOSSARY
-----------
  Atomic            one indivisible step from every other thread's view
  Critical section  code region protected by a lock
  Data race         unsynchronized access, at least one write
  Deadlock          goroutines wait for each other forever
  Hoare triple      {P} f {Q}: if P holds before f, Q holds after
  Rely-guarantee    adds R (assumed interference) and G (promised
                    preservation) to the triple
  TOCTOU            time-of-check to time-of-use
  Linearizability   concurrent behaviour equals some one-at-a-time ordering
  Porcupine         Go library that checks histories for linearizability
  Ablation          switch off one part to measure its contribution
  Held-out set      evaluation data untouched until the final measurement
