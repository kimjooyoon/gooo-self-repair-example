# Single-recorder policy

This repository is an independent, append-only self-repair execution example.

- `kimjooyoon/gooo-self-repair-example` is the only repository this task may write.
- The caller-owned fixture is this repository itself; no source is generated from or applied to `meta-ontology-go`.
- The lifecycle record is append-only. A counterexample may be refuted, but its `BEFORE_REFUTED` history must remain present.
- Generated evaluators must be checked by the immutable parent evaluator and proof-kernel fixtures; a generated evaluator may not prove itself.
- Pull requests are the only place where Go formatting, vetting, testing, building, and conformance execute.
- Do not add aggregate scores, percentages, hidden resets, or closure based only on hashes or replay.
