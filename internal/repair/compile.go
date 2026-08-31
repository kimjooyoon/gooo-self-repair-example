package repair

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Compilation struct {
	IR       SemanticIR
	IRDigest string
}

func Compile(beforePath, afterPath, contractPath, irPath string) (Compilation, error) {
	var contract Contract
	if err := readJSON(contractPath, &contract); err != nil {
		return Compilation{}, err
	}
	if contract.DenominatorCells != 12 || len(contract.Activities) != 12 {
		return Compilation{}, fmt.Errorf("lifecycle contract must contain exactly 12 cells and activities")
	}

	before, err := parseSource(beforePath)
	if err != nil {
		return Compilation{}, err
	}
	after, err := parseSource(afterPath)
	if err != nil {
		return Compilation{}, err
	}
	activities := append(before, after...)
	if len(activities) != contract.DenominatorCells {
		return Compilation{}, fmt.Errorf("released .gooo activity count %d does not equal denominator %d", len(activities), contract.DenominatorCells)
	}
	for index, expected := range contract.Activities {
		actual := activities[index]
		if actual.ActivityID != expected.ActivityID {
			return Compilation{}, fmt.Errorf("activity cell %d is %s, expected %s", expected.Cell, actual.ActivityID, expected.ActivityID)
		}
		if filepath.Base(actual.Source) != expected.Source {
			return Compilation{}, fmt.Errorf("activity %s source is %s, expected %s", actual.ActivityID, actual.Source, expected.Source)
		}
	}

	beforeDigest, err := DigestFile(beforePath)
	if err != nil {
		return Compilation{}, err
	}
	afterDigest, err := DigestFile(afterPath)
	if err != nil {
		return Compilation{}, err
	}
	ir := SemanticIR{
		Schema:             "gooo.self-repair.semantic-ir.v1",
		BeforeSourceDigest: beforeDigest,
		AfterSourceDigest:  afterDigest,
		DenominatorCells:   contract.DenominatorCells,
		Activities:         activities,
	}
	if err := os.MkdirAll(filepath.Dir(irPath), 0o755); err != nil {
		return Compilation{}, err
	}
	if err := writeJSON(irPath, ir); err != nil {
		return Compilation{}, err
	}
	irDigest, err := DigestFile(irPath)
	if err != nil {
		return Compilation{}, err
	}
	return Compilation{IR: ir, IRDigest: irDigest}, nil
}

func parseSource(path string) ([]SourceActivity, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var activities []SourceActivity
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") || fields[0] == "version" {
			continue
		}
		if fields[0] != "activity" || len(fields) < 2 {
			return nil, fmt.Errorf("invalid activity line in %s: %q", path, scanner.Text())
		}
		activity := SourceActivity{ActivityID: fields[1], Source: filepath.Base(path)}
		for _, field := range fields[2:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid activity attribute in %s: %q", path, field)
			}
			switch parts[0] {
			case "phase":
				activity.Phase = parts[1]
			case "input":
				activity.Input = parts[1]
			case "output":
				activity.Output = parts[1]
			default:
				return nil, fmt.Errorf("unknown activity attribute %q", parts[0])
			}
		}
		if activity.Phase == "" || activity.Input == "" || activity.Output == "" {
			return nil, fmt.Errorf("activity %s is incomplete", activity.ActivityID)
		}
		activities = append(activities, activity)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return activities, nil
}

func loadCorpus(path string) (ScenarioCorpus, string, error) {
	var corpus ScenarioCorpus
	if err := readJSON(path, &corpus); err != nil {
		return ScenarioCorpus{}, "", err
	}
	if len(corpus.Cases) != 12 {
		return ScenarioCorpus{}, "", fmt.Errorf("scenario corpus must contain exactly 12 cases")
	}
	digest, err := DigestCanonical(corpus.Cases)
	if err != nil {
		return ScenarioCorpus{}, "", err
	}
	return corpus, digest, nil
}

func verifyIR(path, beforePath, afterPath string) (SemanticIR, string, error) {
	var ir SemanticIR
	if err := readJSON(path, &ir); err != nil {
		return SemanticIR{}, "", err
	}
	beforeDigest, err := DigestFile(beforePath)
	if err != nil {
		return SemanticIR{}, "", err
	}
	afterDigest, err := DigestFile(afterPath)
	if err != nil {
		return SemanticIR{}, "", err
	}
	if ir.BeforeSourceDigest != beforeDigest || ir.AfterSourceDigest != afterDigest || len(ir.Activities) != 12 || ir.DenominatorCells != 12 {
		return SemanticIR{}, "", fmt.Errorf("semantic IR does not bind the two sources and 12-cell contract")
	}
	digest, err := DigestFile(path)
	return ir, digest, err
}
