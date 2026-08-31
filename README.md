# gooo-self-repair-example

An executable, append-only example of a self-repair loop with a fixed counterexample.

The example compiles two `.gooo` sources into semantic IR, runs a generated BEFORE and AFTER evaluator over exactly twelve cases, and emits a human-readable report plus machine-readable receipts. The BEFORE evaluator intentionally contains the meaning bug: an unknown top-level decision is treated as `FIXED_POINT`. The AFTER evaluator accepts only the explicit `FIXED_POINT` decision and otherwise returns `FAIL_CLOSED` with `FEEDBACK_COVERAGE_DECISION_UNKNOWN`.

Three BEFORE counterexamples are retained as historical `REFUTED` cases while their append-only lifecycle continues through `CANDIDATE`, `INDEPENDENT_EVALUATION`, `AUTHORIZED_ADOPTION`, and `AFTER_CLOSED`. The generated evaluator is bound to immutable parent evaluator and proof-kernel fixtures, so it cannot establish its own correctness.

All Go commands run in GitHub Actions only. The conformance job uses a caller-owned temporary fixture and never writes to another repository.
