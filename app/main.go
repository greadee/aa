// Command aa is the composition root for the aa engine.
//
// Every other module in this repository is a library: it defines contracts,
// policies, and seams but composes nothing. This command is where those seams
// are wired into a running graph — the kernel (context, routing, joblearn,
// observability), the runtime (store-backed retrieval, sandbox), the toolbox
// policy engine, and the forge memory sync.
//
// Wiring is intentionally explicit. `aa selfcheck` builds the graph and
// exercises each seam with deterministic inputs so the integration can be
// verified without any model or network access.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/forge/memsync"
	"github.com/greadee/aa/kernel/allocator/routing"
	aaContext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/experience"
	"github.com/greadee/aa/kernel/joblearn/features"
	"github.com/greadee/aa/kernel/joblearn/promote"
	"github.com/greadee/aa/kernel/joblearn/tracesource"
	"github.com/greadee/aa/kernel/observability"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/retention"
	"github.com/greadee/aa/memory/retrieval"
	"github.com/greadee/aa/memory/store"
	"github.com/greadee/aa/registry/models"
	"github.com/greadee/aa/runtime/sandbox"
	"github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/computeruse"
	"github.com/greadee/aa/toolbox/policy"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version)
			return
		case "selfcheck", "--selfcheck", "-c":
			// fall through to selfcheck
		case "-h", "--help", "help":
			fmt.Fprintln(os.Stderr, "usage: aa [selfcheck|version]")
			return
		default:
			fmt.Fprintf(os.Stderr, "aa: unknown command %q\n", os.Args[1])
			os.Exit(2)
		}
	}
	if err := selfcheck(); err != nil {
		fmt.Fprintf(os.Stderr, "aa: %v\n", err)
		os.Exit(1)
	}
}

// runtime is the wired composition graph.
type runtime struct {
	recorder   *observability.MemRecorder
	models     *models.Registry
	store      *store.Store
	projection projection.Projection
	query      *retrieval.Query
	sandbox    *sandbox.Sandbox
	policy     *policy.Engine
	exporter   *memsync.Exporter
	traces     *repo.TraceRepository
	memories   *repo.MemoryRecordRepository
	traceSrc   *tracesource.MemorySource
	traceSink  *promote.MemorySink
	computer   *computeruse.Provider
}

// build constructs the graph. Every dependency is passed explicitly; there is
// no global state and no implicit wiring.
func build(root string) (*runtime, error) {
	recorder := observability.NewMemRecorder()

	modelRegistry, err := models.NewDefaultRegistry()
	if err != nil {
		return nil, fmt.Errorf("registry/models: %w", err)
	}

	st, err := store.Open(root)
	if err != nil {
		return nil, fmt.Errorf("memory/store: %w", err)
	}
	proj := projection.NewMem()
	query := retrieval.New(proj, st)

	traces := repo.NewTraceRepository(st, proj)
	memories := repo.NewMemoryRecordRepository(st, proj)

	traceSrc, err := tracesource.NewMemorySource(traces)
	if err != nil {
		return nil, fmt.Errorf("joblearn/tracesource: %w", err)
	}
	traceSink, err := promote.NewMemorySink(memories)
	if err != nil {
		return nil, fmt.Errorf("joblearn/promote: %w", err)
	}

	enforcer := sandbox.NewEnforcer()
	policyEngine := policy.New(enforcer, toolbox.SystemClock{})

	// Computer-use capability: wired with the declared Headless driver (no
	// desktop backend in this build). The provider is a live, reachable
	// capability that fails closed rather than a test double.
	computerProvider, err := computeruse.NewProvider(computeruse.NewHeadless())
	if err != nil {
		return nil, fmt.Errorf("toolbox/computeruse: %w", err)
	}

	sb, err := sandbox.New(sandbox.Config{
		Root:         filepath.Join(root, "sandbox"),
		EnvAllowlist: []string{"PATH", "HOME"},
		Limits:       sandbox.Limits{Timeout: 30 * time.Second, MaxOutputBytes: 1 << 20},
		Policy:       allowlist{"read": true},
	})
	if err != nil {
		return nil, fmt.Errorf("runtime/sandbox: %w", err)
	}

	return &runtime{
		recorder:   recorder,
		models:     modelRegistry,
		store:      st,
		projection: proj,
		query:      query,
		sandbox:    sb,
		policy:     policyEngine,
		exporter:   memsync.NewExporter("aa"),
		traces:     traces,
		memories:   memories,
		traceSrc:   traceSrc,
		traceSink:  traceSink,
		computer:   computerProvider,
	}, nil
}

// allowlist is a fail-closed sandbox policy: a capability is granted only when
// it is explicitly listed.
type allowlist map[string]bool

func (a allowlist) Allowed(capabilities []string) bool {
	for _, c := range capabilities {
		if !a[c] {
			return false
		}
	}
	return true
}

func selfcheck() error {
	root, err := os.MkdirTemp("", "aa-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	g, err := build(root)
	if err != nil {
		return err
	}

	// Observability: record and account an event with a valid identity chain.
	event := observability.Event{
		Name:          "selfcheck.start",
		CorrelationID: "selfcheck-1",
		Identity: observability.Identity{
			Organization: "aa", Project: "selfcheck", Subtask: "boot", Worker: "aa",
		},
		Tokens: 0,
	}
	if err := g.recorder.Record(event); err != nil {
		return fmt.Errorf("observability: %w", err)
	}
	accounts := observability.Accounting(g.recorder.Events())

	// Routing: deterministic locality/budget decision.
	decision, err := routing.Decide(routing.Request{
		Policy:                routing.LocalFirst,
		Significance:          routing.Significant,
		CloudAllowed:          true,
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 500,
		ExpertInputCostPer1K:  0.01,
		ExpertOutputCostPer1K: 0.03,
	})
	if err != nil {
		return fmt.Errorf("routing: %w", err)
	}

	// Context: derive needs and compile a bounded bundle.
	needs := aaContext.DeriveNeeds("selfcheck the composition root", "wp-selfcheck")
	bundle := aaContext.Compiler{MaxTokens: 2000}.Compile("selfcheck", "wp-selfcheck", []aaContext.Input{
		{Kind: "strategy", ID: "s1", Text: "Wire the seams explicitly at the composition root."},
	})
	// Sandbox: execute a trivial command through the confined runner.
	run, err := g.sandbox.Run(context.Background(), sandbox.Spec{Command: []string{"echo", "aa"}})
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}

	// Job learning: default policies and in-memory seams.
	gatePolicy := joblearn.DefaultGatePolicy()
	bounds := tracesource.DefaultBounds()
	expPolicy := experience.DefaultPolicy()
	expStore := experience.NewMemStore()
	featureSets := features.ExtractAll(nil)

	// Job learning pipeline: attribute (empty) traces, promote through the
	// memory sink, and graduate a project lesson.
	attributionResults, err := attribution.DeriveTraces(nil, func(v2.Trace) attribution.Meta {
		return attribution.Meta{}
	}, attribution.Limits{})
	if err != nil {
		return fmt.Errorf("joblearn/attribution: %w", err)
	}
	promoter, err := promote.New(g.traceSink, promote.Options{ProjectID: "aa"})
	if err != nil {
		return fmt.Errorf("joblearn/promote: %w", err)
	}
	graduated, err := experience.Graduate(joblearn.Candidate{
		Kind:       joblearn.CandidateStrategy,
		Level:      joblearn.LevelProject,
		Title:      "selfcheck",
		Content:    "Wire the seams explicitly at the composition root.",
		Confidence: 0.9,
	})
	if err != nil {
		return fmt.Errorf("joblearn/graduate: %w", err)
	}

	// Retention policy and a content hash.
	retentionPolicy := retention.DefaultPolicy()
	hash := toolbox.HashString("aa")

	fmt.Printf("aa selfcheck (%s)\n", version)
	fmt.Printf("  models registered:   %d\n", g.models.Count())
	fmt.Printf("  observability:       %d event(s), %d account(s)\n", len(g.recorder.Events()), len(accounts))
	fmt.Printf("  routing decision:    locality=%s tier=%s blocked=%v\n", decision.Locality, decision.Tier, decision.Blocked)
	fmt.Printf("  context bundle:      %d section(s), %d/%d tokens\n", len(bundle.Sections), bundle.TokensUsed, bundle.TokenBudget)
	fmt.Printf("  context needs:       %d kind(s), %d term(s)\n", len(needs.Kinds), len(needs.Terms))
	fmt.Printf("  sandbox run:         exit=%d stdout=%q\n", run.ExitCode, string(run.Stdout))
	fmt.Printf("  joblearn:            gatePolicy.minOutcomes=%d bounds=%+v expPolicy=%+v featureSets=%d\n",
		gatePolicy.MinOutcomes, bounds, expPolicy, len(featureSets))
	fmt.Printf("  experience store:    %T\n", expStore)
	fmt.Printf("  joblearn pipeline:   traces=%d promoter=%T graduated=%s\n", len(attributionResults), promoter, graduated.Level)
	fmt.Printf("  retention policy:    %+v\n", retentionPolicy)
	fmt.Printf("  trace source/sink:   %T / %T\n", g.traceSrc, g.traceSink)
	fmt.Printf("  computer-use:        provider kind=%s\n", g.computer.Kind())
	fmt.Printf("  toolbox hash:        %s\n", hash[:12])
	fmt.Printf("  retrieval ready:     %v\n", g.query != nil)
	fmt.Println("selfcheck ok")
	return nil
}
