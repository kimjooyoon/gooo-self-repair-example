package repair

const (
	StateClosed  = "CLOSED"
	StateRefuted = "REFUTED"
	StateUnknown = "UNKNOWN"
)

type Contract struct {
	Schema           string             `json:"schema"`
	DenominatorCells int                `json:"denominator_cells"`
	Activities       []ContractActivity `json:"activities"`
}

type ContractActivity struct {
	Cell       int    `json:"cell"`
	ActivityID string `json:"activity_id"`
	Source     string `json:"source"`
}

type SourceActivity struct {
	ActivityID string `json:"activity_id"`
	Phase      string `json:"phase"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	Source     string `json:"source"`
}

type SemanticIR struct {
	Schema             string           `json:"schema"`
	BeforeSourceDigest string           `json:"before_source_digest"`
	AfterSourceDigest  string           `json:"after_source_digest"`
	DenominatorCells   int              `json:"denominator_cells"`
	Activities         []SourceActivity `json:"activities"`
}

type ScenarioCorpus struct {
	Schema string     `json:"schema"`
	Cases  []Scenario `json:"cases"`
}

type Scenario struct {
	CaseID             string `json:"case_id"`
	Phase              string `json:"phase"`
	Kind               string `json:"kind"`
	BeforeDecision     string `json:"before_decision"`
	AfterDecision      string `json:"after_decision"`
	Truth              string `json:"truth"`
	CandidatePatchID   string `json:"candidate_patch_id"`
	ExpectedFinalState string `json:"expected_final_state"`
}

type Transition struct {
	Sequence int    `json:"sequence"`
	State    string `json:"state"`
	Reason   string `json:"reason"`
}

type CaseResult struct {
	CaseID                    string       `json:"case_id"`
	Phase                     string       `json:"phase"`
	Kind                      string       `json:"kind"`
	BeforeDecision            string       `json:"before_decision"`
	BeforeEvaluatorOutcome    string       `json:"before_evaluator_outcome"`
	AfterDecision             string       `json:"after_decision"`
	AfterEvaluatorOutcome     string       `json:"after_evaluator_outcome"`
	AfterFeedbackCode         string       `json:"after_feedback_code,omitempty"`
	BeforeMeaningBugExposed   bool         `json:"before_meaning_bug_exposed"`
	ExpectedRefutationReceipt bool         `json:"expected_refutation_receipt"`
	FinalState                string       `json:"final_state"`
	Lifecycle                 []Transition `json:"lifecycle"`
}

type ClaimRecord struct {
	RecordType                string       `json:"record_type"`
	CaseID                    string       `json:"case_id"`
	Phase                     string       `json:"phase"`
	Kind                      string       `json:"kind"`
	Sequence                  int          `json:"sequence"`
	FinalState                string       `json:"final_state"`
	BeforeOutcome             string       `json:"before_outcome"`
	AfterOutcome              string       `json:"after_outcome"`
	ExpectedRefutationReceipt bool         `json:"expected_refutation_receipt"`
	Lifecycle                 []Transition `json:"lifecycle"`
	RecordDigest              string       `json:"record_digest"`
}

type CounterexampleRecord struct {
	RecordType       string       `json:"record_type"`
	CaseID           string       `json:"case_id"`
	PreviousDigest   string       `json:"previous_record_digest"`
	ObservedDecision string       `json:"observed_decision"`
	BeforeOutcome    string       `json:"before_outcome"`
	ExpectedOutcome  string       `json:"expected_outcome"`
	Feedback         string       `json:"feedback"`
	Lifecycle        []Transition `json:"lifecycle"`
	RecordDigest     string       `json:"record_digest"`
}
