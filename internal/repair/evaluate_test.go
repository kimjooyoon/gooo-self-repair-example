package repair

import (
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-self-repair-example/generated"
)

func TestRequiredLifecycle(t *testing.T) {
	lifecycle := lifecycleFor(Scenario{Kind: "COUNTEREXAMPLE"})
	if !hasRequiredLifecycle(lifecycle) {
		t.Fatal("counterexample lifecycle is not append-only")
	}
}

func TestArtifactContractNames(t *testing.T) {
	if len(artifactNames) != artifactCount {
		t.Fatalf("artifact count = %d, want %d", len(artifactNames), artifactCount)
	}
	if filepath.Base(artifactNames[0]) != "repair-manifest.json" {
		t.Fatalf("first artifact = %q", artifactNames[0])
	}
}

func TestGeneratedEvaluatorBoundary(t *testing.T) {
	if generated.BeforeEvaluate("UNKNOWN").Outcome != StateClosed {
		t.Fatal("BEFORE regression fixture no longer exposes the historical bug")
	}
	after := generated.AfterEvaluate("UNKNOWN")
	if after.Outcome != "FAIL_CLOSED" || after.FeedbackCode != "FEEDBACK_COVERAGE_DECISION_UNKNOWN" {
		t.Fatal("AFTER evaluator did not fail closed")
	}
}
