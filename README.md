# interleaveguard tasks

Task 1: Setup and baseline (about 2-3 days)
Install Go, and get go test -race and Porcupine running.
Write one tiny bank-account module with the check-then-act bug (a withdraw that checks the balance under a lock, releases it, then updates under a lock).
Write a normal test and run it with -race. It should pass, which shows the bug is invisible to the race detector.
Write one manual failing test: use a sync.WaitGroup or channel barrier so two goroutines both pass the check before either updates. Now it fails every time.
That manual test is exactly what your tool will later generate automatically, so this is your reference.

Easy first split: you do the Go example and the failing test, Vedhant sets up the LLM API access and a repo skeleton.

Task 2: Benchmark (about 4-5 days)
Build 15-20 small Go modules, each about 30-80 lines.
Seed 10-12 buggy ones covering the types in the brief: atomicity violation, TOCTOU, deadlock (lock-order inversion), lost update, and check-then-act (two workers claim the same job).
Add 5-8 clean controls. Make some look suspicious but be correctly synchronized, otherwise your false-positive rate means nothing.
For each one, write down the intended behavior and the bug in a short ground-truth file. You’ll score the tool against this later, and you’ll need it for the “≥70% specs judged correct” criterion too.
Make sure each bug is reproducible with a forced schedule.
