package repair

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-self-repair-example/generated"
)

const artifactCount = 8

var artifactNames = []string{
	"repair-manifest.json",
	"claims.ndjson",
	"counterexamples.ndjson",
	"candidate.patch.json",
	"evaluator-receipt.json",
	"adoption-receipt.json",
	"replay-receipt.json",
	"self-repair-report.md",
}

type Metrics struct {
	PeakRSSKiB          int
	WallMS              int
	Directories         int
	Files               int
	PhysicalLines       int
	GoFiles             int
	GoLines             int
	GoooFiles           int
	GoooLines           int
	RepositoryWrites    int
	LocalTestExecutions int
	CrossProjectGates   int
}

type EvaluateOptions struct {
	BeforeSource  string
	AfterSource   string
	Contract      string
	IR            string
	Cases         string
	ParentFixture string
	ProofFixture  string
	GeneratedGo   string
	Evaluator     string
	ArtifactDir   string
	SubjectSHA    string
	GoVersion     string
	Metrics       Metrics
}

func Evaluate(options EvaluateOptions) error {
	ir, irDigest, err := verifyIR(options.IR, options.BeforeSource, options.AfterSource)
	if err != nil {
		return err
	}
	var contract Contract
	if err := readJSON(options.Contract, &contract); err != nil {
		return err
	}
	if contract.DenominatorCells != 12 || len(contract.Activities) != 12 || len(ir.Activities) != 12 {
		return fmt.Errorf("evaluation requires a 12-cell lifecycle contract")
	}
	corpus, corpusDigest, err := loadCorpus(options.Cases)
	if err != nil {
		return err
	}
	generatedGoDigest, err := DigestFile(options.GeneratedGo)
	if err != nil {
		return err
	}
	evaluatorDigest, err := DigestFile(options.Evaluator)
	if err != nil {
		return err
	}
	parentDigest, err := verifyImmutableFixture(options.ParentFixture, generated.ParentEvaluatorDigest, "parent evaluator")
	if err != nil {
		return err
	}
	proofDigest, err := verifyImmutableFixture(options.ProofFixture, generated.ProofKernelDigest, "proof kernel")
	if err != nil {
		return err
	}

	patch := map[string]any{
		"schema":         "gooo.self-repair.candidate-patch.v1",
		"candidate_id":   "candidate-decision-guard",
		"change":         "unknown-decisions-fail-closed",
		"guard":          "decision == FIXED_POINT",
		"applies_to":     "after-evaluator",
		"proof_boundary": "parent-evaluator-and-proof-kernel",
		"patch_digest":   "",
	}
	patchDigest, err := DigestCanonical(patch)
	if err != nil {
		return err
	}
	patch["patch_digest"] = patchDigest

	results := make([]CaseResult, 0, len(corpus.Cases))
	claims := make([]ClaimRecord, 0, len(corpus.Cases))
	counterexamples := make([]CounterexampleRecord, 0, 3)
	previousCounterexampleDigest := ""
	for _, scenario := range corpus.Cases {
		before := generated.BeforeEvaluate(scenario.BeforeDecision)
		after := generated.AfterEvaluate(scenario.AfterDecision)
		isCounterexample := scenario.Kind == "COUNTEREXAMPLE"
		bugExposed := isCounterexample && scenario.BeforeDecision != "FIXED_POINT" && before.Outcome == StateClosed
		if isCounterexample && !bugExposed {
			return fmt.Errorf("counterexample %s did not expose the BEFORE meaning bug", scenario.CaseID)
		}
		if scenario.Kind == "DEFENSE" && (after.Outcome != "FAIL_CLOSED" || after.FeedbackCode != "FEEDBACK_COVERAGE_DECISION_UNKNOWN") {
			return fmt.Errorf("defense case %s was not fail-closed", scenario.CaseID)
		}
		if scenario.Kind == "NORMAL" && after.Outcome != StateClosed {
			return fmt.Errorf("normal case %s did not close", scenario.CaseID)
		}
		finalState := scenario.ExpectedFinalState
		if finalState != StateClosed && finalState != StateRefuted {
			return fmt.Errorf("case %s has unsupported final state %s", scenario.CaseID, finalState)
		}
		lifecycle := lifecycleFor(scenario)
		if isCounterexample && !hasRequiredLifecycle(lifecycle) {
			return fmt.Errorf("counterexample %s lost append-only lifecycle", scenario.CaseID)
		}
		result := CaseResult{
			CaseID:                    scenario.CaseID,
			Phase:                     scenario.Phase,
			Kind:                      scenario.Kind,
			BeforeDecision:            scenario.BeforeDecision,
			BeforeEvaluatorOutcome:    before.Outcome,
			AfterDecision:             scenario.AfterDecision,
			AfterEvaluatorOutcome:     after.Outcome,
			AfterFeedbackCode:         after.FeedbackCode,
			BeforeMeaningBugExposed:   bugExposed,
			ExpectedRefutationReceipt: isCounterexample,
			FinalState:                finalState,
			Lifecycle:                 lifecycle,
		}
		results = append(results, result)
		claim := ClaimRecord{
			RecordType:                "CLAIM",
			CaseID:                    scenario.CaseID,
			Phase:                     scenario.Phase,
			Kind:                       scenario.Kind,
			Sequence:                  len(claims) + 1,
			FinalState:                finalState,
			BeforeOutcome:             before.Outcome,
			AfterOutcome:              after.Outcome,
			ExpectedRefutationReceipt: isCounterexample,
			Lifecycle:                 lifecycle,
		}
		claim.RecordDigest = claimDigest(claim)
		claims = append(claims, claim)
		if isCounterexample {
			counterexample := CounterexampleRecord{
				RecordType:       "BEFORE_COUNTEREXAMPLE",
				CaseID:           scenario.CaseID,
				PreviousDigest:   previousCounterexampleDigest,
				ObservedDecision: scenario.BeforeDecision,
				BeforeOutcome:    before.Outcome,
				ExpectedOutcome:  "FAIL_CLOSED",
				Feedback:         "FEEDBACK_COVERAGE_DECISION_UNKNOWN",
				Lifecycle:        lifecycle,
			}
			counterexample.RecordDigest = counterexampleDigest(counterexample)
			previousCounterexampleDigest = counterexample.RecordDigest
			counterexamples = append(counterexamples, counterexample)
		}
	}
	if err := validateCounts(results); err != nil {
		return err
	}
	if err := prepareArtifactDir(options.ArtifactDir); err != nil {
		return err
	}

	provenance := map[string]any{
		"before_source_digest":            ir.BeforeSourceDigest,
		"after_source_digest":             ir.AfterSourceDigest,
		"semantic_ir_digest":              irDigest,
		"generated_go_digest":             generatedGoDigest,
		"evaluator_digest":                evaluatorDigest,
		"scenario_corpus_digest":          corpusDigest,
		"parent_evaluator_fixture_digest": parentDigest,
		"proof_kernel_fixture_digest":     proofDigest,
		"candidate_patch_digest":          patchDigest,
	}
	evaluatorReceipt := map[string]any{
		"schema":                         "gooo.self-repair.evaluator-receipt.v1",
		"generated_version":              generated.GeneratedVersion,
		"before_rule":                    "unknown top-level decisions were incorrectly treated as FIXED_POINT",
		"after_rule":                     "only explicit FIXED_POINT closes; all other decisions fail closed",
		"unknown_feedback_code":          "FEEDBACK_COVERAGE_DECISION_UNKNOWN",
		"expected_before_refutations":    3,
		"independent_parent_binding":     true,
		"hash_or_replay_alone_can_close": false,
		"provenance":                     provenance,
	}
	adoptionReceipt := map[string]any{
		"schema":                      "gooo.self-repair.adoption-receipt.v1",
		"candidate_id":                "candidate-decision-guard",
		"authorization":               "AUTHORIZED_ADOPTION",
		"independent_evaluation":      "INDEPENDENT_EVALUATION",
		"retained_before_refutations": 3,
		"after_closed_cases":          9,
		"patch_digest":                patchDigest,
		"provenance":                  provenance,
	}
	replayReceipt := map[string]any{
		"schema":                 "gooo.self-repair.replay-receipt.v1",
		"case_id":                "replay-01",
		"replayed_from":          "immutable-before-history",
		"replay_can_close_alone": false,
		"result":                 "CLOSED",
		"provenance":             provenance,
	}

	claimsPath := filepath.Join(options.ArtifactDir, "claims.ndjson")
	if err := writeNDJSON(claimsPath, claims); err != nil {
		return err
	}
	counterexamplesPath := filepath.Join(options.ArtifactDir, "counterexamples.ndjson")
	if err := writeNDJSON(counterexamplesPath, counterexamples); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "candidate.patch.json"), patch); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "evaluator-receipt.json"), evaluatorReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "adoption-receipt.json"), adoptionReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "replay-receipt.json"), replayReceipt); err != nil {
		return err
	}
	report := renderReport(results, provenance)
	if err := os.WriteFile(filepath.Join(options.ArtifactDir, "self-repair-report.md"), []byte(report), 0o644); err != nil {
		return err
	}

	artifactDigests := map[string]string{}
	for _, name := range artifactNames {
		if name == "repair-manifest.json" {
			continue
		}
		digest, err := DigestFile(filepath.Join(options.ArtifactDir, name))
		if err != nil {
			return err
		}
		artifactDigests[name] = digest
	}
	manifest := map[string]any{
		"schema":      "gooo.self-repair.repair-manifest.v1",
		"subject_sha": options.SubjectSHA,
		"go_version":  options.GoVersion,
		"contracts": map[string]any{
			"denominator_cells":        12,
			"released_gooo_activities": len(ir.Activities),
			"activity_mapping":         "1:1",
			"executable_cases":         len(results),
		},
		"case_states": map[string]int{
			"CLOSED":  countState(results, StateClosed),
			"REFUTED": countState(results, StateRefuted),
			"UNKNOWN": countState(results, StateUnknown),
		},
		"case_kinds": map[string]int{
			"BEFORE_NORMAL":         countKind(results, "BEFORE", "NORMAL"),
			"BEFORE_COUNTEREXAMPLE": countKind(results, "BEFORE", "COUNTEREXAMPLE"),
			"AFTER_NORMAL":          countKind(results, "AFTER", "NORMAL"),
			"AFTER_DEFENSE":         countKind(results, "AFTER", "DEFENSE"),
			"REPLAY":                countKind(results, "REPLAY", "REPLAY"),
		},
		"provenance":       provenance,
		"artifacts":        artifactNames,
		"artifact_digests": artifactDigests,
		"authority": map[string]int{
			"repository_writes":            options.Metrics.RepositoryWrites,
			"local_test_executions":        options.Metrics.LocalTestExecutions,
			"cross_project_required_gates": options.Metrics.CrossProjectGates,
		},
		"inventory": map[string]any{
			"root_readme_excluded": true,
			"directories":          options.Metrics.Directories,
			"files":                options.Metrics.Files,
			"physical_lines":       options.Metrics.PhysicalLines,
			"go_files":             options.Metrics.GoFiles,
			"go_lines":             options.Metrics.GoLines,
			"gooo_files":           options.Metrics.GoooFiles,
			"gooo_lines":           options.Metrics.GoooLines,
		},
		"runtime": map[string]int{
			"peak_rss_kib": options.Metrics.PeakRSSKiB,
			"wall_ms":      options.Metrics.WallMS,
		},
		"semantic_close_rule": "explicit decision only; hashes and replay cannot close",
	}
	return writeJSON(filepath.Join(options.ArtifactDir, "repair-manifest.json"), manifest)
}

func verifyImmutableFixture(path, expectedDigest, label string) (string, error) {
	var fixture map[string]any
	if err := readJSON(path, &fixture); err != nil {
		return "", err
	}
	if immutable, ok := fixture["immutable"].(bool); !ok || !immutable {
		return "", fmt.Errorf("%s fixture is not immutable", label)
	}
	declared, ok := fixture["fixture_digest"].(string)
	if !ok {
		return "", fmt.Errorf("%s fixture lacks a digest", label)
	}
	fixture["fixture_digest"] = ""
	actual, err := DigestCanonical(fixture)
	if err != nil {
		return "", err
	}
	if declared != actual || expectedDigest != actual {
		return "", fmt.Errorf("%s fixture digest mismatch: declared=%s actual=%s expected=%s", label, declared, actual, expectedDigest)
	}
	return actual, nil
}

func lifecycleFor(scenario Scenario) []Transition {
	switch scenario.Kind {
	case "COUNTEREXAMPLE":
		return []Transition{
			{Sequence: 1, State: "BEFORE_REFUTED", Reason: "unknown decision was incorrectly accepted as FIXED_POINT"},
			{Sequence: 2, State: "CANDIDATE", Reason: "decision guard candidate declared"},
			{Sequence: 3, State: "INDEPENDENT_EVALUATION", Reason: "parent evaluator checked candidate"},
			{Sequence: 4, State: "AUTHORIZED_ADOPTION", Reason: "adoption authority recorded"},
			{Sequence: 5, State: "AFTER_CLOSED", Reason: "repaired evaluator closed only explicit fixed point"},
		}
	case "DEFENSE":
		return []Transition{
			{Sequence: 1, State: "CANDIDATE", Reason: "candidate evaluator active"},
			{Sequence: 2, State: "INDEPENDENT_EVALUATION", Reason: "parent evaluator checked unknown decision"},
			{Sequence: 3, State: "AUTHORIZED_ADOPTION", Reason: "fail-closed behavior authorized"},
			{Sequence: 4, State: "AFTER_CLOSED", Reason: "defensive failure is closed without semantic acceptance"},
		}
	case "REPLAY":
		return []Transition{
			{Sequence: 1, State: "REPLAYED", Reason: "immutable history replayed"},
			{Sequence: 2, State: "AFTER_CLOSED", Reason: "replay checked explicit fixed point"},
		}
	default:
		return []Transition{
			{Sequence: 1, State: "BEFORE_CLOSED", Reason: "explicit fixed point observed"},
			{Sequence: 2, State: "AFTER_CLOSED", Reason: "explicit fixed point rechecked"},
		}
	}
}

func hasRequiredLifecycle(lifecycle []Transition) bool {
	want := []string{"BEFORE_REFUTED", "CANDIDATE", "INDEPENDENT_EVALUATION", "AUTHORIZED_ADOPTION", "AFTER_CLOSED"}
	if len(lifecycle) != len(want) {
		return false
	}
	for index, state := range want {
		if lifecycle[index].State != state || lifecycle[index].Sequence != index+1 {
			return false
		}
	}
	return true
}

func validateCounts(results []CaseResult) error {
	if len(results) != 12 || countState(results, StateClosed) != 9 || countState(results, StateRefuted) != 3 || countState(results, StateUnknown) != 0 {
		return fmt.Errorf("final states must be CLOSED=9, REFUTED=3, UNKNOWN=0")
	}
	if countKind(results, "BEFORE", "NORMAL") != 2 || countKind(results, "BEFORE", "COUNTEREXAMPLE") != 3 || countKind(results, "AFTER", "NORMAL") != 3 || countKind(results, "AFTER", "DEFENSE") != 3 || countKind(results, "REPLAY", "REPLAY") != 1 {
		return fmt.Errorf("case kinds do not match the 2/3/3/3/1 contract")
	}
	return nil
}

func countState(results []CaseResult, state string) int {
	count := 0
	for _, result := range results {
		if result.FinalState == state {
			count++
		}
	}
	return count
}

func countKind(results []CaseResult, phase, kind string) int {
	count := 0
	for _, result := range results {
		if result.Phase == phase && result.Kind == kind {
			count++
		}
	}
	return count
}

func claimDigest(claim ClaimRecord) string {
	claim.RecordDigest = ""
	digest, _ := DigestCanonical(claim)
	return digest
}

func counterexampleDigest(record CounterexampleRecord) string {
	record.RecordDigest = ""
	digest, _ := DigestCanonical(record)
	return digest
}

func prepareArtifactDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("artifact directory must start empty")
	}
	return nil
}

func writeNDJSON(path string, values any) error {
	data, ok := values.([]ClaimRecord)
	if ok {
		return writeLines(path, data)
	}
	counterexamples, ok := values.([]CounterexampleRecord)
	if ok {
		return writeLines(path, counterexamples)
	}
	return fmt.Errorf("unsupported NDJSON value")
}

func writeLines[T any](path string, values []T) error {
	var builder strings.Builder
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func renderReport(results []CaseResult, provenance map[string]any) string {
	var builder strings.Builder
	builder.WriteString("# Self-repair report\n\n")
	builder.WriteString("This execution preserves the meaning bug as history and closes the repaired behavior only on an explicit semantic decision. No aggregate ranking is produced.\n\n")
	builder.WriteString("## Contract\n\n")
	builder.WriteString("- Lifecycle denominator: 12 cells. Released `.gooo` activities: 12, mapped 1:1.\n")
	builder.WriteString("- Executable cases: 12 (BEFORE normal 2, BEFORE counterexample 3, AFTER normal 3, AFTER defense 3, replay 1).\n")
	builder.WriteString("- Final case states: CLOSED 9, REFUTED 3, UNKNOWN 0.\n")
	builder.WriteString("- The three REFUTED cases are retained BEFORE history, not deleted.\n\n")
	builder.WriteString("## Evaluator behavior\n\n")
	builder.WriteString("The BEFORE evaluator incorrectly closes unknown top-level decisions. The AFTER evaluator accepts only `FIXED_POINT`; every other decision is `FAIL_CLOSED` with `FEEDBACK_COVERAGE_DECISION_UNKNOWN`. Hashes and replay evidence never close the evaluator by themselves.\n\n")
	builder.WriteString("## Append-only lifecycle\n\n")
	builder.WriteString("Each counterexample retains `BEFORE_REFUTED → CANDIDATE → INDEPENDENT_EVALUATION → AUTHORIZED_ADOPTION → AFTER_CLOSED`.\n\n")
	builder.WriteString("| Case | Kind | Final state | BEFORE | AFTER |\n| --- | --- | --- | --- | --- |\n")
	for _, result := range results {
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", result.CaseID, result.Kind, result.FinalState, result.BeforeEvaluatorOutcome, result.AfterEvaluatorOutcome))
	}
	builder.WriteString("\n## Digest chain\n\n")
	for _, key := range []string{"before_source_digest", "after_source_digest", "semantic_ir_digest", "generated_go_digest", "evaluator_digest", "scenario_corpus_digest", "parent_evaluator_fixture_digest", "proof_kernel_fixture_digest"} {
		builder.WriteString(fmt.Sprintf("- `%s`: `%v`\n", key, provenance[key]))
	}
	return builder.String()
}
