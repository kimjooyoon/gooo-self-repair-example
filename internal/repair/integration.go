package repair

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-self-repair-example/generated"
)

const integrationSchema = "gooo/self-repair/integration-input/v1"

var integrationArtifactNames = []string{
	"release-input-receipt.json",
	"candidate-receipt.json",
	"mutation-receipt.json",
	"frontier-receipt.json",
	"oracle-receipt.json",
	"drift-receipt.json",
	"experience-receipt.json",
	"claims.ndjson",
	"cycle-receipt.json",
	"self-repair-report.md",
	"repair-manifest.json",
}

type UnknownCoordinate struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type ReleaseEvidence struct {
	ID                       string `json:"id"`
	Repository               string `json:"repository"`
	Tag                      string `json:"tag"`
	ReleaseAPIPath           string `json:"release_api_path"`
	ReleaseID                int64  `json:"release_id"`
	ReleaseNodeID            string `json:"release_node_id"`
	TagObjectSHA             string `json:"tag_object_sha"`
	TargetCommitSHA          string `json:"target_commit_sha"`
	ReleaseAPIIdentityDigest string `json:"release_api_identity_digest"`
	ArtifactKind             string `json:"artifact_kind"`
	ArtifactName             string `json:"artifact_name"`
	ArtifactURL              string `json:"artifact_url"`
	ArtifactDigest           string `json:"artifact_digest"`
	ObservedArtifactDigest   string `json:"observed_artifact_digest"`
	ArtifactPath             string `json:"artifact_path"`
	Immutable                bool   `json:"immutable"`
}

type ObservationEvidence struct {
	Decision       string `json:"decision"`
	ObservedState  string `json:"observed_state"`
	ExpectedState  string `json:"expected_state"`
	SourceLine     int    `json:"source_line"`
	ArtifactDigest string `json:"artifact_digest"`
}

type ProposalEvidence struct {
	State          string            `json:"state"`
	CandidateCount int               `json:"candidate_count"`
	CandidateID    string            `json:"candidate_id"`
	Unknown        UnknownCoordinate `json:"unknown"`
	ArtifactDigest string            `json:"artifact_digest"`
}

type MutationEvidence struct {
	Generated           int    `json:"generated"`
	Attempted           int    `json:"attempted"`
	Killed              int    `json:"killed"`
	Unknown             int    `json:"unknown"`
	Refuted             int    `json:"refuted"`
	SelectedMutantID    string `json:"selected_mutant_id"`
	SelectedMutantState string `json:"selected_mutant_state"`
	ArtifactDigest      string `json:"artifact_digest"`
}

type SelectionEvidence struct {
	State          string            `json:"state"`
	Decision       string            `json:"decision"`
	SelectedID     string            `json:"selected_id"`
	Unknown        UnknownCoordinate `json:"unknown"`
	ArtifactDigest string            `json:"artifact_digest"`
}

type ExecutionCounts struct {
	Total       int `json:"total"`
	Executed    int `json:"executed"`
	Reused      int `json:"reused"`
	Skipped     int `json:"skipped"`
	NotObserved int `json:"not_observed"`
}

type FrontierEvidence struct {
	State          string            `json:"state"`
	CaseID         string            `json:"case_id"`
	Reason         string            `json:"reason"`
	Tests          ExecutionCounts   `json:"tests"`
	Unknown        UnknownCoordinate `json:"unknown"`
	ArtifactDigest string            `json:"artifact_digest"`
}

type ReuseEvidence struct {
	Decision       string          `json:"decision"`
	PlanStatus     string          `json:"plan_status"`
	Authorized     bool            `json:"authorized"`
	ActualReused   int             `json:"actual_reused"`
	Tests          ExecutionCounts `json:"tests"`
	ArtifactDigest string          `json:"artifact_digest"`
}

type OracleEvidence struct {
	State          string `json:"state"`
	CaseID         string `json:"case_id"`
	Decision       string `json:"decision"`
	Reason         string `json:"reason"`
	ArtifactDigest string `json:"artifact_digest"`
}

type DriftEvidence struct {
	Decision       string `json:"decision"`
	Reason         string `json:"reason"`
	ArtifactDigest string `json:"artifact_digest"`
}

type ExperienceEvidence struct {
	BeforeState              string `json:"before_state"`
	AfterState               string `json:"after_state"`
	SelectedCandidateID      string `json:"selected_candidate_id"`
	KnownRecurrencesBefore   int    `json:"known_recurrences_before"`
	KnownRecurrencesAfter    int    `json:"known_recurrences_after"`
	AvoidedRefutedCandidates int    `json:"avoided_refuted_candidates"`
	ReplayComparisons        int    `json:"replay_comparisons"`
	ReplayMismatches         int    `json:"replay_mismatches"`
	ArtifactDigest           string `json:"artifact_digest"`
}

type UtilitySnapshot struct {
	MemoryKib         int `json:"memory_kib"`
	BuildWallMS       int `json:"build_wall_ms"`
	TestWallMS        int `json:"test_wall_ms"`
	ConformanceWallMS int `json:"conformance_wall_ms"`
}

type UtilityPair struct {
	Before         UtilitySnapshot `json:"before"`
	After          UtilitySnapshot `json:"after"`
	SourceDigest   string          `json:"source_digest"`
	Decision       string          `json:"decision"`
	DecisionReason string          `json:"decision_reason"`
}

type IntegrationMetrics struct {
	BeforeUtility     UtilitySnapshot `json:"before_utility"`
	AfterUtility      UtilitySnapshot `json:"after_utility"`
	BuildWallMS       int             `json:"build_wall_ms"`
	TestWallMS        int             `json:"test_wall_ms"`
	ConformanceWallMS int             `json:"conformance_wall_ms"`
	PeakRSSKiB        int             `json:"peak_rss_kib"`
	Tests             ExecutionCounts `json:"tests"`
	GoFiles           int             `json:"go_files"`
	GoLines           int             `json:"go_lines"`
	GoooFiles         int             `json:"gooo_files"`
	GoooLines         int             `json:"gooo_lines"`
	RepositoryWrites  int             `json:"repository_writes"`
}

type IntegrationInput struct {
	Schema           string              `json:"schema"`
	DenominatorCells int                 `json:"denominator_cells"`
	StateCounts      map[string]int      `json:"state_counts"`
	Precedence       []string            `json:"precedence"`
	Releases         []ReleaseEvidence   `json:"releases"`
	Observation      ObservationEvidence `json:"observation"`
	Proposal         ProposalEvidence    `json:"proposal"`
	Mutation         MutationEvidence    `json:"mutation"`
	Selection        SelectionEvidence   `json:"selection"`
	Frontier         FrontierEvidence    `json:"frontier"`
	Reuse            ReuseEvidence       `json:"reuse"`
	Oracle           OracleEvidence      `json:"oracle"`
	OracleNegative   OracleEvidence      `json:"oracle_negative"`
	Drift            DriftEvidence       `json:"drift"`
	DriftNegative    DriftEvidence       `json:"drift_negative"`
	Experience       ExperienceEvidence  `json:"experience"`
	Utility          UtilityPair         `json:"utility"`
	Metrics          IntegrationMetrics  `json:"metrics"`
}

type IntegrationClaim struct {
	Sequence    int                `json:"sequence"`
	ClaimID     string             `json:"claim_id"`
	Stage       string             `json:"stage"`
	State       string             `json:"state"`
	Reason      string             `json:"reason"`
	Evidence    []string           `json:"evidence"`
	Unknown     *UnknownCoordinate `json:"unknown,omitempty"`
	Historical  bool               `json:"historical"`
	SecondCycle bool               `json:"second_cycle"`
}

func Integrate(options EvaluateOptions, inputPath string) error {
	if options.SubjectSHA == "" || options.GoVersion != "1.27.x" {
		return fmt.Errorf("integration must be bound to a subject revision and Go 1.27.x")
	}
	ir, irDigest, err := verifyIR(options.IR, options.BeforeSource, options.AfterSource)
	if err != nil {
		return err
	}
	var contract Contract
	if err := readJSON(options.Contract, &contract); err != nil {
		return err
	}
	if contract.DenominatorCells != 12 || len(contract.Activities) != 12 || len(ir.Activities) != 12 {
		return fmt.Errorf("integration requires an exact 12-cell semantic contract")
	}
	var input IntegrationInput
	if err := readJSON(inputPath, &input); err != nil {
		return err
	}
	if err := validateIntegrationInput(input); err != nil {
		return err
	}
	if options.Metrics.RepositoryWrites != 0 {
		return fmt.Errorf("integration repository writes must remain zero")
	}
	if generated.AfterEvaluate("FIXED_POINT").Outcome != StateClosed || generated.AfterEvaluate("UNKNOWN_TOP_LEVEL").Outcome != "FAIL_CLOSED" {
		return fmt.Errorf("repaired evaluator direct semantic rule is not fail-closed")
	}
	if input.Observation.ObservedState != StateClosed || input.Observation.ExpectedState != StateUnknown {
		return fmt.Errorf("observation did not preserve the historical meaning defect")
	}
	if input.Proposal.State != StateUnknown || input.Proposal.CandidateCount != 1 || input.Proposal.CandidateID != "utility/utility-record-missing" || !validUnknown(input.Proposal.Unknown) {
		return fmt.Errorf("proposer evidence did not produce the bounded UNKNOWN candidate")
	}
	if input.Mutation.Generated != 12 || input.Mutation.Attempted != 12 || input.Mutation.Killed != 10 || input.Mutation.Unknown != 1 || input.Mutation.Refuted != 1 || input.Mutation.SelectedMutantState != "KILLED" {
		return fmt.Errorf("mutation evidence does not match the bounded 12-mutant run")
	}
	if input.Selection.State != StateUnknown || input.Selection.Decision != "INCOMPARABLE_UNKNOWN" || !validUnknown(input.Selection.Unknown) {
		return fmt.Errorf("selector evidence must preserve the crossing-axis UNKNOWN")
	}
	if input.Frontier.State != StateUnknown || !validUnknown(input.Frontier.Unknown) || !validExecutionCounts(input.Frontier.Tests) {
		return fmt.Errorf("frontier evidence must preserve the unobserved execution UNKNOWN")
	}
	if input.Reuse.Decision != StateClosed || input.Reuse.PlanStatus != StateClosed || !input.Reuse.Authorized || input.Reuse.ActualReused != 0 {
		return fmt.Errorf("reuse evidence did not close the exact plan-only judgment")
	}
	if input.Oracle.State != StateClosed || input.Oracle.CaseID != "normal-boundary" || input.Oracle.Decision != StateClosed {
		return fmt.Errorf("independent oracle direct evidence did not close")
	}
	if input.OracleNegative.State != StateRefuted || input.OracleNegative.Decision != StateRefuted {
		return fmt.Errorf("proof-kernel negative control did not remain REFUTED")
	}
	if input.Drift.Decision != StateClosed || input.DriftNegative.Decision != StateRefuted {
		return fmt.Errorf("semantic drift gate evidence is incomplete")
	}
	if input.Experience.BeforeState != StateRefuted || input.Experience.AfterState != StateClosed || input.Experience.SelectedCandidateID == "" || input.Experience.KnownRecurrencesBefore != 1 || input.Experience.KnownRecurrencesAfter != 0 || input.Experience.AvoidedRefutedCandidates != 1 || input.Experience.ReplayComparisons != 2 || input.Experience.ReplayMismatches != 0 {
		return fmt.Errorf("experience memory did not demonstrate second-cycle recurrence prevention")
	}
	if input.Utility.Decision != StateUnknown || input.Utility.DecisionReason == "" || !validUtilitySnapshot(input.Utility.Before) || !validUtilitySnapshot(input.Utility.After) {
		return fmt.Errorf("utility pair must be exact while its cross-axis decision remains UNKNOWN")
	}
	if err := prepareArtifactDir(options.ArtifactDir); err != nil {
		return err
	}

	claims := []IntegrationClaim{
		{Sequence: 1, ClaimID: "independent-oracle-verified", Stage: "INDEPENDENT_ORACLE", State: StateClosed, Reason: "DIRECT_PROOF_KERNEL_RECEIPT_AND_REPAIRED_EVALUATOR_AGREE", Evidence: []string{"proof-kernel-boundary/normal-boundary", "after-evaluator"}},
		{Sequence: 2, ClaimID: "frontier-reuse-judgment", Stage: "TEST_FRONTIER", State: StateClosed, Reason: "EXACT_FRONTIER_AND_IMMUTABLE_REUSE_PLAN_BOUND", Evidence: []string{"test-frontier/normal-alpha", "verification-reuse/valid-reuse-plan"}},
		{Sequence: 3, ClaimID: "second-cycle-recurrence-prevented", Stage: "EXPERIENCE_MEMORY", State: StateClosed, Reason: "KNOWN_REFUTED_CANDIDATE_AVOIDED_ON_SECOND_CYCLE", Evidence: []string{"experience-memory/after", "replay-comparisons=2"}, SecondCycle: true},
		{Sequence: 4, ClaimID: "proposal-exact-utility-gap", Stage: "OBSERVATION_TO_PROPOSAL", State: StateUnknown, Reason: input.Proposal.Unknown.Reason, Evidence: []string{"improvement-proposer/unknown-missing-utility"}, Unknown: &input.Proposal.Unknown},
		{Sequence: 5, ClaimID: "utility-cross-axis-selection", Stage: "EVIDENCE_FIRST_SELECTION", State: StateUnknown, Reason: input.Selection.Unknown.Reason, Evidence: []string{"improvement-selector/unknown-cross-axis"}, Unknown: &input.Selection.Unknown},
		{Sequence: 6, ClaimID: "affected-test-not-observed", Stage: "IMPACT_FRONTIER", State: StateUnknown, Reason: input.Frontier.Unknown.Reason, Evidence: []string{"test-frontier/unknown-unobserved-execution"}, Unknown: &input.Frontier.Unknown},
		{Sequence: 7, ClaimID: "bounded-mutation-negative-control", Stage: "BOUNDED_SEMANTIC_MUTATION", State: StateRefuted, Reason: "MUTATION_LAB_RETAINED_ONE_INVALID_MUTANT", Evidence: []string{"semantic-mutation-lab/MUT-12-REFUTATION-PRECEDENCE"}, Historical: true},
		{Sequence: 8, ClaimID: "proof-kernel-override-negative-control", Stage: "INDEPENDENT_ORACLE", State: StateRefuted, Reason: input.OracleNegative.Reason, Evidence: []string{"proof-kernel-boundary/refuted-verdict-override"}, Historical: true},
		{Sequence: 9, ClaimID: "semantic-drift-negative-control", Stage: "DRIFT_GATE", State: StateRefuted, Reason: input.DriftNegative.Reason, Evidence: []string{"semantic-drift-guard/refuted-authority-escalation"}, Historical: true},
	}
	if err := validateClaimCounts(claims); err != nil {
		return err
	}
	provenance := map[string]any{
		"before_source_digest":   ir.BeforeSourceDigest,
		"after_source_digest":    ir.AfterSourceDigest,
		"semantic_ir_digest":     irDigest,
		"candidate_patch":        input.Proposal.CandidateID,
		"selected_mutant":        input.Mutation.SelectedMutantID,
		"selected_candidate":     input.Experience.SelectedCandidateID,
		"external_release_count": len(input.Releases),
	}

	releaseReceipt := map[string]any{
		"schema":                  "gooo/self-repair/release-input-receipt/v1",
		"authority":               "github-release-api",
		"fixed_by_api_and_digest": true,
		"releases":                input.Releases,
		"source":                  filepath.ToSlash(inputPath),
	}
	candidateReceipt := map[string]any{
		"schema":                  "gooo/self-repair/candidate-receipt/v1",
		"observation":             input.Observation,
		"proposal":                input.Proposal,
		"candidate_state":         StateUnknown,
		"candidate_is_not_closed": true,
		"reason":                  "EXACT_UTILITY_EVIDENCE_IS_MISSING_FROM_PROPOSER_INPUT",
	}
	mutationReceipt := map[string]any{
		"schema":          "gooo/self-repair/mutation-receipt/v1",
		"mutation":        input.Mutation,
		"bounded":         true,
		"precedence":      input.Precedence,
		"selected_change": "unknown top-level decision is no longer treated as FIXED_POINT",
	}
	frontierReceipt := map[string]any{
		"schema":        "gooo/self-repair/frontier-receipt/v1",
		"frontier":      input.Frontier,
		"reuse":         input.Reuse,
		"utility":       input.Utility,
		"utility_state": StateUnknown,
		"reason":        "EXACT_PAIR_IS_PRESENT_BUT_RESOURCE_AXES_CROSS_FOR_UNWEIGHTED_SELECTION",
	}
	oracleReceipt := map[string]any{
		"schema":                         "gooo/self-repair/oracle-receipt/v1",
		"direct_positive":                input.Oracle,
		"direct_negative":                input.OracleNegative,
		"repaired_evaluator":             map[string]any{"explicit_fixed_point": StateClosed, "unknown_top_level": "FAIL_CLOSED"},
		"core_semantic_authority":        StateClosed,
		"core_semantic_authority_reason": "DIRECT_ORACLE_AND_EXPLICIT_AFTER_EVALUATOR_EVIDENCE",
	}
	driftReceipt := map[string]any{
		"schema":                      "gooo/self-repair/drift-gate-receipt/v1",
		"positive":                    input.Drift,
		"negative_control":            input.DriftNegative,
		"decision":                    StateClosed,
		"negative_control_precedence": StateRefuted,
	}
	experienceReceipt := map[string]any{
		"schema":                            "gooo/self-repair/experience-receipt/v1",
		"experience":                        input.Experience,
		"before_cycle":                      map[string]any{"state": input.Experience.BeforeState, "known_refuted_recurrences": input.Experience.KnownRecurrencesBefore},
		"after_cycle":                       map[string]any{"state": input.Experience.AfterState, "known_refuted_recurrences": input.Experience.KnownRecurrencesAfter, "selected_candidate_id": input.Experience.SelectedCandidateID},
		"second_cycle_recurrence_prevented": true,
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "release-input-receipt.json"), releaseReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "candidate-receipt.json"), candidateReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "mutation-receipt.json"), mutationReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "frontier-receipt.json"), frontierReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "oracle-receipt.json"), oracleReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "drift-receipt.json"), driftReceipt); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "experience-receipt.json"), experienceReceipt); err != nil {
		return err
	}
	if err := writeIntegrationClaims(filepath.Join(options.ArtifactDir, "claims.ndjson"), claims); err != nil {
		return err
	}

	cycleReceipt := map[string]any{
		"schema":                  "gooo/self-repair/cycle-receipt/v1",
		"decision":                "CONFORMANCE_CLOSED",
		"selected_candidate":      map[string]any{"id": input.Experience.SelectedCandidateID, "state": StateClosed, "change": "unknown decisions fail closed"},
		"lifecycle":               []string{"OBSERVATION", "GOOO_CANDIDATE", "BOUNDED_SEMANTIC_MUTATION", "IMPACT_TEST_FRONTIER", "VERIFICATION_REUSE", "INDEPENDENT_ORACLE", "EVIDENCE_FIRST_SELECTION", "DRIFT_GATE", "EXPERIENCE_MEMORY", "SECOND_CYCLE"},
		"claims":                  map[string]int{"CLOSED": 3, "UNKNOWN": 3, "REFUTED": 3},
		"precedence":              input.Precedence,
		"unknown_contract":        []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"},
		"core_semantic_authority": StateClosed,
		"external_utility":        map[string]any{"state": StateUnknown, "reason": input.Utility.DecisionReason, "direct_exact_pair": true},
		"provenance":              provenance,
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "cycle-receipt.json"), cycleReceipt); err != nil {
		return err
	}

	report := renderIntegrationReport(input, claims, provenance)
	if err := os.WriteFile(filepath.Join(options.ArtifactDir, "self-repair-report.md"), []byte(report), 0o644); err != nil {
		return err
	}
	artifactDigests := map[string]string{}
	for _, name := range integrationArtifactNames {
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
		"schema":                  "gooo/self-repair/repair-manifest.v2",
		"subject_sha":             options.SubjectSHA,
		"go_version":              options.GoVersion,
		"denominator":             map[string]any{"total": 12, "exact": true, "activity_mapping": "1:1", "released_gooo_activities": len(ir.Activities)},
		"claim_denominator":       map[string]any{"total": 9, "exact": true},
		"state_counts":            map[string]int{"CLOSED": 3, "UNKNOWN": 3, "REFUTED": 3},
		"precedence":              input.Precedence,
		"artifacts":               integrationArtifactNames,
		"artifact_digests":        artifactDigests,
		"external_releases":       input.Releases,
		"metrics":                 input.Metrics,
		"utility_pair":            input.Utility,
		"authority":               map[string]int{"repository_writes": input.Metrics.RepositoryWrites, "local_test_executions": 0, "cross_project_required_gates": 0},
		"core_semantic_authority": map[string]any{"state": StateClosed, "direct_evidence": true},
		"external_utility":        map[string]any{"state": StateUnknown, "direct_exact_pair": true, "selection_blocked_by": input.Utility.DecisionReason},
		"semantic_close_rule":     "explicit decision only; hashes and replay cannot close",
	}
	return writeJSON(filepath.Join(options.ArtifactDir, "repair-manifest.json"), manifest)
}

func validateIntegrationInput(input IntegrationInput) error {
	if input.Schema != integrationSchema || input.DenominatorCells != 12 || len(input.Releases) != 8 {
		return fmt.Errorf("integration input must bind eight releases and the fixed 12-cell denominator")
	}
	if strings.Join(input.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" {
		return fmt.Errorf("integration precedence must be REFUTED,UNKNOWN,CLOSED")
	}
	for _, state := range []string{StateClosed, StateUnknown, StateRefuted} {
		if input.StateCounts[state] != 3 {
			return fmt.Errorf("integration state count %s must be 3", state)
		}
	}
	seen := map[string]bool{}
	expectedIDs := map[string]bool{
		"improvement-proposer":   true,
		"semantic-mutation-lab": true,
		"improvement-selector":   true,
		"proof-kernel-boundary":  true,
		"test-frontier":          true,
		"verification-reuse":     true,
		"semantic-drift-guard":   true,
		"experience-memory":      true,
	}
	for _, release := range input.Releases {
		if release.ID == "" || !expectedIDs[release.ID] || seen[release.ID] || release.Repository == "" || release.Tag == "" || release.ReleaseAPIPath == "" || release.ReleaseID <= 0 || release.ReleaseNodeID == "" || release.TagObjectSHA == "" || release.TargetCommitSHA == "" || !validDigest(release.ReleaseAPIIdentityDigest) || release.ArtifactKind == "" || release.ArtifactName == "" || release.ArtifactURL == "" || !validDigest(release.ArtifactDigest) || release.ObservedArtifactDigest != release.ArtifactDigest || !release.Immutable {
			return fmt.Errorf("release evidence is incomplete or not digest-bound: %s", release.ID)
		}
		seen[release.ID] = true
	}
	if len(seen) != len(expectedIDs) {
		return fmt.Errorf("release evidence is missing one or more required immutable inputs")
	}
	if !validExecutionCounts(input.Metrics.Tests) || input.Metrics.BuildWallMS < 0 || input.Metrics.TestWallMS < 0 || input.Metrics.ConformanceWallMS < 0 || input.Metrics.PeakRSSKiB <= 0 || input.Metrics.GoFiles <= 0 || input.Metrics.GoLines <= 0 || input.Metrics.GoooFiles <= 0 || input.Metrics.GoooLines <= 0 {
		return fmt.Errorf("CI integer metrics are incomplete")
	}
	return nil
}

func validUnknown(unknown UnknownCoordinate) bool {
	return unknown.Stage != "" && unknown.Step != "" && unknown.Reason != "" && unknown.UnknownClass != "" && unknown.NextOperation != "" && unknown.BlockedBy != nil
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func validUtilitySnapshot(snapshot UtilitySnapshot) bool {
	return snapshot.MemoryKib >= 0 && snapshot.BuildWallMS >= 0 && snapshot.TestWallMS >= 0 && snapshot.ConformanceWallMS >= 0
}

func validExecutionCounts(counts ExecutionCounts) bool {
	return counts.Total >= 0 && counts.Executed >= 0 && counts.Reused >= 0 && counts.Skipped >= 0 && counts.NotObserved >= 0 && counts.Total == counts.Executed+counts.Reused+counts.Skipped+counts.NotObserved
}

func validateClaimCounts(claims []IntegrationClaim) error {
	if len(claims) != 9 {
		return fmt.Errorf("integration must emit exactly nine claims")
	}
	counts := map[string]int{}
	for _, claim := range claims {
		counts[claim.State]++
		if claim.State == StateUnknown && (claim.Unknown == nil || !validUnknown(*claim.Unknown)) {
			return fmt.Errorf("UNKNOWN claim %s lacks all six coordinates", claim.ClaimID)
		}
	}
	for _, state := range []string{StateClosed, StateUnknown, StateRefuted} {
		if counts[state] != 3 {
			return fmt.Errorf("claim state %s count is %d, want 3", state, counts[state])
		}
	}
	return nil
}

func writeIntegrationClaims(path string, claims []IntegrationClaim) error {
	var builder strings.Builder
	for _, claim := range claims {
		data, err := json.Marshal(claim)
		if err != nil {
			return err
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func renderIntegrationReport(input IntegrationInput, claims []IntegrationClaim, provenance map[string]any) string {
	var builder strings.Builder
	builder.WriteString("# Gooo self-repair integration report\n\n")
	builder.WriteString("The repaired evaluator closes only an explicit `FIXED_POINT`. The historical unknown-decision acceptance remains an append-only REFUTED observation, and the second cycle records that the known refuted candidate was avoided.\n\n")
	builder.WriteString("## Fixed contract\n\n")
	fmt.Fprintf(&builder, "- denominator: `12/12` Gooo activities mapped `1:1`\n- claim states: `CLOSED=3`, `UNKNOWN=3`, `REFUTED=3`\n- precedence: `%s`\n- UNKNOWN fields: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, `blocked_by`\n- repository writes: `%d`\n\n", strings.Join(input.Precedence, " > "), input.Metrics.RepositoryWrites)
	builder.WriteString("## Closed lifecycle\n\n")
	builder.WriteString("`OBSERVATION → GOOO_CANDIDATE → BOUNDED_SEMANTIC_MUTATION → IMPACT_TEST_FRONTIER → VERIFICATION_REUSE → INDEPENDENT_ORACLE → EVIDENCE_FIRST_SELECTION → DRIFT_GATE → EXPERIENCE_MEMORY → SECOND_CYCLE`\n\n")
	fmt.Fprintf(&builder, "Selected candidate: `%s`, second-cycle state: **%s**, known REFUTED recurrence: `%d → %d`.\n\n", input.Experience.SelectedCandidateID, input.Experience.AfterState, input.Experience.KnownRecurrencesBefore, input.Experience.KnownRecurrencesAfter)
	builder.WriteString("## Claim matrix\n\n| # | claim | state | reason |\n|---:|---|---|---|\n")
	for _, claim := range claims {
		fmt.Fprintf(&builder, "| %d | %s | %s | %s |\n", claim.Sequence, claim.ClaimID, claim.State, claim.Reason)
	}
	builder.WriteString("\n## Exact integer observations\n\n")
	fmt.Fprintf(&builder, "| metric | value |\n|---|---:|\n| build_wall_ms | %d |\n| test_wall_ms | %d |\n| conformance_wall_ms | %d |\n| peak_rss_kib | %d |\n| test_total | %d |\n| test_executed | %d |\n| test_reused | %d |\n| test_skipped | %d |\n| test_not_observed | %d |\n| Go files | %d |\n| Go lines | %d |\n| Gooo files | %d |\n| Gooo lines | %d |\n| repository_writes | %d |\n", input.Metrics.BuildWallMS, input.Metrics.TestWallMS, input.Metrics.ConformanceWallMS, input.Metrics.PeakRSSKiB, input.Metrics.Tests.Total, input.Metrics.Tests.Executed, input.Metrics.Tests.Reused, input.Metrics.Tests.Skipped, input.Metrics.Tests.NotObserved, input.Metrics.GoFiles, input.Metrics.GoLines, input.Metrics.GoooFiles, input.Metrics.GoooLines, input.Metrics.RepositoryWrites)
	builder.WriteString("\n## Exact utility pair\n\n")
	fmt.Fprintf(&builder, "`%s`: memory `%d → %d`, build `%d → %d`, test `%d → %d`, conformance `%d → %d`; decision remains **%s** because `%s`.\n\n", input.Utility.SourceDigest, input.Utility.Before.MemoryKib, input.Utility.After.MemoryKib, input.Utility.Before.BuildWallMS, input.Utility.After.BuildWallMS, input.Utility.Before.TestWallMS, input.Utility.After.TestWallMS, input.Utility.Before.ConformanceWallMS, input.Utility.After.ConformanceWallMS, input.Utility.Decision, input.Utility.DecisionReason)
	builder.WriteString("## Provenance\n\n")
	for _, key := range []string{"before_source_digest", "after_source_digest", "semantic_ir_digest", "candidate_patch", "selected_mutant", "selected_candidate"} {
		fmt.Fprintf(&builder, "- `%s`: `%v`\n", key, provenance[key])
	}
	builder.WriteString("\nExternal utility is not CLOSED without a non-crossing exact pair; release API identity and asset digests are checked before any receipt is trusted.\n")
	return builder.String()
}
