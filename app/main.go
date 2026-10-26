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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/greadee/aa/app/internal/controlplane"
	"github.com/greadee/aa/app/internal/obsplane"
	"github.com/greadee/aa/app/internal/syncplane"
	"github.com/greadee/aa/app/internal/visplane"
	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/forge/memsync"
	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/allocator/routing"
	"github.com/greadee/aa/kernel/api"
	aaContext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/experience"
	"github.com/greadee/aa/kernel/joblearn/features"
	"github.com/greadee/aa/kernel/joblearn/promote"
	"github.com/greadee/aa/kernel/joblearn/tracesource"
	"github.com/greadee/aa/kernel/observability"
	"github.com/greadee/aa/kernel/scheduler"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/retention"
	"github.com/greadee/aa/memory/retrieval"
	"github.com/greadee/aa/memory/store"
	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/registry/models"
	"github.com/greadee/aa/registry/policies"
	"github.com/greadee/aa/registry/routines"
	"github.com/greadee/aa/registry/teams"
	"github.com/greadee/aa/runtime/lifecycle"
	"github.com/greadee/aa/runtime/sandbox"
	"github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/computeruse"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
	uicp "github.com/greadee/aa/ui/controlplane"
	"github.com/greadee/aa/ui/surfaces/tui"
	wsurface "github.com/greadee/aa/ui/surfaces/web"
	visretention "github.com/greadee/aa/visualizer/retention"
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
		case "serve":
			if err := serve(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "aa: %v\n", err)
				os.Exit(1)
			}
			return
		case "console":
			if err := console(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "aa: %v\n", err)
				os.Exit(1)
			}
			return
		case "-h", "--help", "help":
			fmt.Fprintln(os.Stderr, "usage: aa [serve|console|selfcheck|version]")
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

// serve runs the control plane over loopback until interrupted.
func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:7100", "loopback listen address")
	token := fs.String("token", os.Getenv("AA_CONTROL_TOKEN"), "bearer token required on all paths except /health")
	root := fs.String("root", defaultRoot(), "state root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := build(*root)
	if err != nil {
		return err
	}
	handler := controlplane.New(controlplane.Config{
		Version:      version,
		Token:        *token,
		Status:       g.status,
		Backend:      &backend{g: g},
		Observations: observationBackend{plane: g.obs},
		Visualizer:   visualizerBackend{plane: g.vis},
		Registry:     definitionsBackend{g: g},
		Lifecycle:    lifecycleBackend{manager: g.life},
	})
	srv := &http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stderr, "aa control plane listening on http://%s%s\n", *addr, controlplane.APIPrefix)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
	return nil
}

func defaultRoot() string {
	if v := os.Getenv("AA_ROOT"); v != "" {
		return v
	}
	return ".aa"
}

// console runs the console over the control plane: the read-only web surface
// (--web) or the text surface. It imports only the ui client, never modules.
func console(args []string) error {
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	control := fs.String("control", "http://127.0.0.1:7100/api/v1", "control-plane base URL")
	token := fs.String("token", os.Getenv("AA_CONTROL_TOKEN"), "bearer token for the control plane")
	project := fs.String("project", "default", "project id")
	web := fs.Bool("web", false, "serve the read-only web surface instead of the text surface")
	fake := fs.Bool("fake", false, "use the deterministic fake control plane (offline; no server)")
	addr := fs.String("addr", "127.0.0.1:7101", "web listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var client uicp.Client
	if *fake {
		client = uicp.NewFake()
	} else {
		client = uicp.NewHTTPClient(*control, *token)
	}
	if *web {
		srv := &http.Server{Addr: *addr, Handler: wsurface.New(client, *project).Handler(), ReadHeaderTimeout: 5 * time.Second}
		fmt.Fprintf(os.Stderr, "aa console web on http://%s\n", *addr)
		errCh := make(chan error, 1)
		go func() { errCh <- srv.ListenAndServe() }()
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}
		return nil
	}
	return tui.New(client, *project).Run(context.Background(), os.Stdin, os.Stdout)
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
	tools      *registry.Registry
	graph      *planner.Graph
	sched      *scheduler.Scheduler
	api        *api.Service
	syncer     *syncplane.Plane
	obs        *obsplane.Plane
	vis        *visplane.Plane
	life       *lifecycle.Manager
	teams      *teams.Registry
	routines   *routines.Registry
	policies   *policies.Registry
}

// status reports the aggregate host status for the control plane.
func (g *runtime) status() controlplane.Status {
	projects := 0
	if recs, err := g.store.ListRecords("project_record"); err == nil {
		projects = len(recs)
	}
	nodes := 0
	if g.syncer != nil {
		nodes = len(g.syncer.Nodes())
	}
	return controlplane.Status{Projects: projects, ActiveAssignments: 0, Nodes: nodes}
}

// backend implements controlplane.Backend over the composed graph.
type backend struct {
	g   *runtime
	mu  sync.Mutex
	seq int
}

func (b *backend) Projects() []controlplane.Project {
	recs, err := b.g.store.ListRecords("project_record")
	if err != nil {
		return nil
	}
	out := make([]controlplane.Project, 0, len(recs))
	for _, r := range recs {
		var m struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(r.Data, &m)
		out = append(out, controlplane.Project{ID: r.ID, Name: m.Name})
	}
	return out
}

func (b *backend) CreateProject(id, name string) error {
	if id == "" {
		return errors.New("project id is required")
	}
	data, err := json.Marshal(map[string]any{
		"contractVersion": "2.0",
		"kind":            "project_record",
		"id":              id,
		"name":            name,
		"state":           "CREATED",
		"layoutVersion":   "1",
	})
	if err != nil {
		return err
	}
	rec, err := store.NewRecord("project_record", id, 1, data)
	if err != nil {
		return err
	}
	if _, err = b.g.store.PutRecord(rec); err != nil {
		return err
	}
	b.emit("project:"+id, protocol.Event{
		SourceType: protocol.SourceSession,
		Confidence: protocol.ConfidenceObserved,
		Action:     "project.create",
		Actor:      "control-plane",
		Metadata:   map[string]string{"status": "created"},
	})
	return nil
}

func (b *backend) WorkPackages(projectID string) []controlplane.WorkPackageView {
	if projectID != b.g.graph.ProjectID {
		return nil
	}
	wps := b.g.api.Status().WorkPackages
	out := make([]controlplane.WorkPackageView, 0, len(wps))
	for _, wp := range wps {
		out = append(out, controlplane.WorkPackageView{ID: wp.ID, Title: wp.Title, State: string(wp.State), DependsOn: wp.DependsOn})
	}
	return out
}

func (b *backend) SubmitWork(projectID, title string, dependsOn []string) (string, error) {
	if projectID != b.g.graph.ProjectID {
		return "", fmt.Errorf("unknown project %q", projectID)
	}
	b.mu.Lock()
	b.seq++
	id := fmt.Sprintf("wp-%d", b.seq)
	b.mu.Unlock()
	if err := b.g.graph.Add(planner.WorkPackage{ID: id, Title: title, DependsOn: dependsOn}); err != nil {
		return "", err
	}
	b.emit("project:"+projectID, protocol.Event{
		SourceType:    protocol.SourceWorkDelta,
		Confidence:    protocol.ConfidenceExact,
		Action:        "work.submit",
		Actor:         "control-plane",
		WorkPackageID: id,
		Metadata:      map[string]string{"status": "submitted"},
	})
	return id, nil
}

func (b *backend) Assignments() []controlplane.AssignmentView {
	as := b.g.api.Status().Assignments
	out := make([]controlplane.AssignmentView, 0, len(as))
	for _, a := range as {
		out = append(out, controlplane.AssignmentView{ID: a.ID, WorkPackageID: a.WorkPackageID, WorkerID: a.WorkerID, State: a.State, Attempt: a.Attempt})
	}
	return out
}

func (b *backend) Approve(gateID string) error {
	b.g.api.Approve(gateID)
	b.emit("control-plane", protocol.Event{
		SourceType:    protocol.SourceSession,
		Confidence:    protocol.ConfidenceObserved,
		Action:        "gate.approve",
		Actor:         "console",
		WorkPackageID: gateID,
		Metadata:      map[string]string{"status": "approved"},
	})
	return nil
}

func (b *backend) records(kind string) []json.RawMessage {
	recs, err := b.g.store.ListRecords(kind)
	if err != nil {
		return nil
	}
	out := make([]json.RawMessage, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Data)
	}
	return out
}

func (b *backend) Issues(string) []json.RawMessage     { return b.records("issue") }
func (b *backend) Strategies(string) []json.RawMessage { return b.records("strategy") }

func (b *backend) Events(string) []json.RawMessage {
	evs, err := b.g.store.Events()
	if err != nil {
		return nil
	}
	out := make([]json.RawMessage, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.Data)
	}
	return out
}

func (b *backend) Nodes() []controlplane.Node {
	if b.g.syncer == nil {
		return nil
	}
	nodes := b.g.syncer.Nodes()
	out := make([]controlplane.Node, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, controlplane.Node{
			ID:        string(n.ID),
			Name:      n.Name,
			Authority: n.Local,
			State:     n.State,
		})
	}
	return out
}

func (b *backend) SyncStatus() controlplane.SyncStatus {
	st := controlplane.SyncStatus{
		Nodes:     []controlplane.SyncNode{},
		Transfers: []controlplane.SyncTransfer{},
	}
	if b.g.syncer == nil {
		return st
	}
	syncStatus := b.g.syncer.Status(context.Background())
	st.LocalNode = string(syncStatus.LocalNode)
	for _, n := range syncStatus.Nodes {
		st.Nodes = append(st.Nodes, controlplane.SyncNode{
			ID: string(n.ID), Name: n.Name, Local: n.Local, State: n.State,
		})
	}
	for _, t := range syncStatus.Transfers {
		created := ""
		if !t.CreatedAt.IsZero() {
			created = t.CreatedAt.UTC().Format(time.RFC3339Nano)
		}
		st.Transfers = append(st.Transfers, controlplane.SyncTransfer{
			TransferID:    string(t.TransferID),
			NodeID:        string(t.NodeID),
			WorkPackageID: t.WorkPackageID,
			ArtifactID:    t.ArtifactID,
			State:         string(t.State),
			Hash:          t.Hash,
			Bytes:         t.Bytes,
			CreatedAt:     created,
		})
	}
	return st
}

func (b *backend) Tools() []controlplane.Tool {
	if b.g.tools == nil {
		return nil
	}
	ms := b.g.tools.List("")
	out := make([]controlplane.Tool, 0, len(ms))
	for _, m := range ms {
		out = append(out, controlplane.Tool{ID: string(m.ID), Name: m.Name, Kind: m.ToolKind, Sandbox: m.Sandbox != nil})
	}
	return out
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

	teamRegistry := teams.NewRegistry()
	routineRegistry := routines.NewRegistry()
	policyRegistry := policies.NewRegistry()
	if err := loadDefinitions(st, teamRegistry, routineRegistry, policyRegistry); err != nil {
		return nil, err
	}

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

	// Toolbox registry: register the builtin computer-use tool so /tools has a
	// real manifest to serve.
	toolReg := registry.New()
	if err := toolReg.Register(computeruse.Manifest("tool_screen")); err != nil {
		return nil, fmt.Errorf("toolbox/registry: %w", err)
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

	// Sync: one local node identity plus the composed transport seam. v1 is
	// single-node (D6); the console's nodes read model is served from here and
	// no remote execution is exposed.
	seed, err := syncplane.SeedFromRoot(root)
	if err != nil {
		return nil, err
	}
	syncer := syncplane.New(syncplane.Config{Name: localNodeName(), Seed: seed})

	// Observation: one owner, durable per-session journals under the state
	// root. Retention bounds only the projected report view.
	observations, err := obsplane.New(obsplane.Config{
		Owner:     "app",
		Dir:       filepath.Join(root, "obsv"),
		Retention: journal.Policy{MaxEvents: 1000},
	})
	if err != nil {
		return nil, err
	}

	// Visualizer: replay/live over the same obsv transport, canonical session
	// browsing over memory, and a retained projected trail.
	visualizerPlane, err := visplane.New(visplane.Config{
		Observations:     observations.Server(),
		ObservedSessions: observations.Sessions,
		Memory:           query,
		Retention:        visretention.Policy{MaxFrames: 500},
	})
	if err != nil {
		return nil, err
	}

	// Worker lifecycle: instantiated and observed on the scheduler dispatch
	// path. Execution stays disabled, so dispatch still runs worker.Disabled.
	lifecycleManager := lifecycle.New(lifecycle.Config{
		Sink: lifecycleObserver{plane: observations},
	})

	// Kernel control path: one project graph + scheduler, exposed in-process by
	// kernel/api. Execution stays disabled on the core path (no model, no worker).
	graph := planner.NewGraph("default", "plan-main")
	sched, err := scheduler.New(graph, scheduler.Config{
		Registry:  role_allocator.New(),
		Lifecycle: lifecycleManager,
	})
	if err != nil {
		return nil, fmt.Errorf("kernel/scheduler: %w", err)
	}
	apiSvc, err := api.New(api.Config{Scheduler: sched})
	if err != nil {
		return nil, fmt.Errorf("kernel/api: %w", err)
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
		tools:      toolReg,
		graph:      graph,
		sched:      sched,
		api:        apiSvc,
		syncer:     syncer,
		obs:        observations,
		vis:        visualizerPlane,
		life:       lifecycleManager,
		teams:      teamRegistry,
		routines:   routineRegistry,
		policies:   policyRegistry,
	}, nil
}

func localNodeName() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "local"
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
	defer g.obs.Close()

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

	// Observation: journal a sandbox seam event and derive its report.
	if !g.obs.Emit("selfcheck", protocol.Event{
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceExact,
		Action:     "sandbox.run",
		Actor:      "aa",
		Metadata:   map[string]string{"tool": "sandbox", "status": "ok"},
	}) {
		return fmt.Errorf("obsv: emit was rejected")
	}
	obsReport, err := g.obs.Report("selfcheck")
	if err != nil {
		return fmt.Errorf("obsv: %w", err)
	}
	visSessions, err := g.vis.Sessions(context.Background())
	if err != nil {
		return fmt.Errorf("visualizer sessions: %w", err)
	}
	visGraph, err := g.vis.Graph(context.Background(), "selfcheck")
	if err != nil {
		return fmt.Errorf("visualizer graph: %w", err)
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
	fmt.Printf("  registry:            teams=%d routines=%d policies=%d\n", g.teams.Count(), g.routines.Count(), g.policies.Count())
	fmt.Printf("  lifecycle:           workers=%d events=%d\n", len(g.life.Workers()), len(g.life.Events()))
	fmt.Printf("  sync:                node=%s nodes=%d\n", g.syncer.LocalID(), len(g.syncer.Nodes()))
	fmt.Printf("  observability:       %d event(s), %d account(s)\n", len(g.recorder.Events()), len(accounts))
	fmt.Printf("  routing decision:    locality=%s tier=%s blocked=%v\n", decision.Locality, decision.Tier, decision.Blocked)
	fmt.Printf("  routing contract:    %+v\n", decision.Contract())
	fmt.Printf("  context bundle:      %d section(s), %d/%d tokens\n", len(bundle.Sections), bundle.TokensUsed, bundle.TokenBudget)
	fmt.Printf("  context needs:       %d kind(s), %d term(s)\n", len(needs.Kinds), len(needs.Terms))
	fmt.Printf("  sandbox run:         exit=%d stdout=%q\n", run.ExitCode, string(run.Stdout))
	fmt.Printf("  observation:         events=%d lastSeq=%d sources=%d\n", obsReport.Events, obsReport.LastSequence, len(obsReport.BySourceType))
	fmt.Printf("  visualizer:          sessions=%d graphNodes=%d graphEdges=%d\n", len(visSessions), len(visGraph.Nodes), len(visGraph.Edges))
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
