package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-self-repair-example/internal/repair"
)

func main() {
	if len(os.Args) < 2 {
		fail("expected compile or evaluate")
	}
	switch os.Args[1] {
	case "compile":
		compile(os.Args[2:])
	case "evaluate":
		evaluate(os.Args[2:])
	default:
		fail("unknown command %q", os.Args[1])
	}
}

func compile(args []string) {
	flags := flag.NewFlagSet("compile", flag.ExitOnError)
	before := flags.String("before-source", "", "BEFORE .gooo source")
	after := flags.String("after-source", "", "AFTER .gooo source")
	contract := flags.String("contract", "", "lifecycle contract")
	ir := flags.String("ir", "", "semantic IR output")
	flags.Parse(args)
	result, err := repair.Compile(*before, *after, *contract, *ir)
	if err != nil {
		fail("compile: %v", err)
	}
	fmt.Println(result.IRDigest)
}

func evaluate(args []string) {
	flags := flag.NewFlagSet("evaluate", flag.ExitOnError)
	options := repair.EvaluateOptions{}
	flags.StringVar(&options.BeforeSource, "before-source", "", "BEFORE .gooo source")
	flags.StringVar(&options.AfterSource, "after-source", "", "AFTER .gooo source")
	flags.StringVar(&options.Contract, "contract", "", "lifecycle contract")
	flags.StringVar(&options.IR, "ir", "", "semantic IR")
	flags.StringVar(&options.Cases, "cases", "", "scenario corpus")
	flags.StringVar(&options.ParentFixture, "parent-fixture", "", "immutable parent evaluator fixture")
	flags.StringVar(&options.ProofFixture, "proof-fixture", "", "immutable proof-kernel fixture")
	flags.StringVar(&options.GeneratedGo, "generated-go", "", "generated evaluator source")
	flags.StringVar(&options.Evaluator, "evaluator", "", "independent evaluator script")
	flags.StringVar(&options.ArtifactDir, "artifact-dir", "", "empty artifact directory")
	flags.StringVar(&options.SubjectSHA, "subject-sha", "", "subject revision")
	flags.StringVar(&options.GoVersion, "go-version", "", "CI Go version")
	bindInt(flags, &options.Metrics.PeakRSSKiB, "peak-rss-kib")
	bindInt(flags, &options.Metrics.WallMS, "wall-ms")
	bindInt(flags, &options.Metrics.Directories, "directories")
	bindInt(flags, &options.Metrics.Files, "files")
	bindInt(flags, &options.Metrics.PhysicalLines, "physical-lines")
	bindInt(flags, &options.Metrics.GoFiles, "go-files")
	bindInt(flags, &options.Metrics.GoLines, "go-lines")
	bindInt(flags, &options.Metrics.GoooFiles, "gooo-files")
	bindInt(flags, &options.Metrics.GoooLines, "gooo-lines")
	bindInt(flags, &options.Metrics.RepositoryWrites, "repository-writes")
	bindInt(flags, &options.Metrics.LocalTestExecutions, "local-test-executions")
	bindInt(flags, &options.Metrics.CrossProjectGates, "cross-project-required-gates")
	flags.Parse(args)
	if err := repair.Evaluate(options); err != nil {
		fail("evaluate: %v", err)
	}
	fmt.Println("self-repair evaluation complete")
}

func bindInt(flags *flag.FlagSet, target *int, name string) {
	flags.IntVar(target, name, 0, name)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
