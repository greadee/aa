// Package distill synthesizes scoped subagent artifacts from trace evidence.
//
// Distillation is deterministic and model-free: the same attributed traces
// always yield the same candidates. Candidates are scoped to a role, trade, or
// task, carry the trace evidence as provenance, and use the existing candidate
// kinds. They are proposals only; memory owns promotion.
package distill

import (
	"fmt"
	"sort"
	"time"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/features"
)

// DefaultSource names the producer recorded in candidate provenance.
const DefaultSource = "aa-kernel/joblearn/distill"

// Record pairs one attributed, trace-derived result with its step features.
type Record struct {
	Result attribution.Result `json:"result"`
	Set    features.Set       `json:"set"`
}

// Policy bounds artifact distillation.
type Policy struct {
	// MinOutcomes is the minimum attributed outcomes for a scope to be learned.
	MinOutcomes int
	// SuccessThreshold is the success rate that yields a strategy artifact.
	SuccessThreshold float64
	// FailureThreshold is the failure rate that yields a pitfall artifact.
	FailureThreshold float64
}

// DefaultPolicy returns conservative thresholds.
func DefaultPolicy() Policy {
	return Policy{MinOutcomes: 3, SuccessThreshold: 0.8, FailureThreshold: 0.5}
}

// Validate checks the policy's bounds.
func (p Policy) Validate() error {
	if p.MinOutcomes < 0 {
		return invalid("minOutcomes must be >= 0")
	}
	for name, v := range map[string]float64{
		"successThreshold": p.SuccessThreshold,
		"failureThreshold": p.FailureThreshold,
	} {
		if v < 0 || v > 1 {
			return invalid("policy.%s = %v is outside [0,1]", name, v)
		}
	}
	return nil
}

// Distiller derives scoped candidate artifacts from trace evidence.
type Distiller struct {
	policy Policy
	source string
	now    func() time.Time
}

// New returns a distiller under a policy.
func New(policy Policy) (*Distiller, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Distiller{policy: policy, source: DefaultSource, now: time.Now}, nil
}

// WithSource overrides the provenance source.
func (d *Distiller) WithSource(source string) *Distiller {
	d.source = source
	return d
}

// WithClock overrides the time source for deterministic tests.
func (d *Distiller) WithClock(now func() time.Time) *Distiller {
	d.now = now
	return d
}

// Distill synthesizes candidates from trace evidence. Results are canonicalized
// first, so the output does not depend on input order.
func (d *Distiller) Distill(records []Record) ([]joblearn.Candidate, error) {
	if err := d.policy.Validate(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	ordered := canonical(records)
	var out []joblearn.Candidate

	out = append(out, d.scopeArtifacts(ordered, attribution.ByRole, joblearn.LevelRole, scopeRole)...)
	out = append(out, d.scopeArtifacts(ordered, attribution.ByTrade, joblearn.LevelRole, scopeTrade)...)
	out = append(out, d.scopeArtifacts(ordered, attribution.ByWorkPackage, joblearn.LevelTask, scopeWorkPackage)...)

	return dedupSort(out), nil
}

// scopeArtifacts emits a strategy or pitfall for each sufficiently evidenced
// scope along one dimension.
func (d *Distiller) scopeArtifacts(records []Record, dimension attribution.Dimension, level joblearn.CandidateLevel, prefix string) []joblearn.Candidate {
	grouped := map[string][]Record{}
	for _, record := range records {
		value := dimensionValue(record.Result.Attribution, dimension)
		if value == "" {
			continue
		}
		grouped[value] = append(grouped[value], record)
	}
	values := make([]string, 0, len(grouped))
	for value := range grouped {
		values = append(values, value)
	}
	sort.Strings(values)

	var out []joblearn.Candidate
	for _, value := range values {
		group := grouped[value]
		if len(group) < d.policy.MinOutcomes {
			continue
		}
		summary := attribution.Summarize(resultsOf(group))
		success := rate(summary.Successes, summary.Outcomes)
		failure := rate(summary.Failures, summary.Outcomes)
		scope := prefix + value

		if success >= d.policy.SuccessThreshold {
			phase := dominantPhase(group)
			out = append(out, d.build(
				joblearn.CandidateStrategy, level, scope,
				fmt.Sprintf("Strategy for %s", value),
				fmt.Sprintf("%d of %d outcomes succeeded; dominant phase %s", summary.Successes, summary.Outcomes, phase),
				group, success,
			))
		}
		if failure >= d.policy.FailureThreshold {
			errorClass := dominantErrorClass(group)
			out = append(out, d.build(
				joblearn.CandidatePitfall, level, scope,
				fmt.Sprintf("Pitfall for %s", value),
				fmt.Sprintf("%d of %d outcomes failed; recurring error class %s", summary.Failures, summary.Outcomes, errorClass),
				group, failure,
			))
		}
	}
	return out
}

func (d *Distiller) build(kind joblearn.CandidateKind, level joblearn.CandidateLevel, scope, title, content string, group []Record, confidence float64) joblearn.Candidate {
	return joblearn.Candidate{
		Kind:       kind,
		Level:      level,
		Scope:      scope,
		Title:      title,
		Content:    content,
		Evidence:   evidenceOf(group),
		Provenance: &joblearn.Provenance{Source: d.source, ProducedAt: d.now().UTC().Format(time.RFC3339)},
		Confidence: clamp01(confidence),
	}
}

// canonical returns records sorted by a total key so grouping and floating
// sums are order-independent.
func canonical(records []Record) []Record {
	ordered := append([]Record(nil), records...)
	sort.SliceStable(ordered, func(i, j int) bool { return recordKey(ordered[i]) < recordKey(ordered[j]) })
	return ordered
}

func recordKey(record Record) string {
	a := record.Result.Attribution
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%v", a.ProjectID, a.WorkPackageID, a.AttemptID, a.Role, a.Trade, a.Worker, a.Sequence, a.Outcome)
}

func resultsOf(group []Record) []attribution.Result {
	out := make([]attribution.Result, len(group))
	for i, record := range group {
		out[i] = record.Result
	}
	return out
}

// dominantPhase returns the most frequent step phase, ties broken lexically.
// It reads phase metadata only, never content.
func dominantPhase(group []Record) string {
	counts := map[string]int{}
	for _, record := range group {
		for _, step := range record.Set.Steps {
			counts[step.Phase]++
		}
	}
	return dominantKey(counts, "none")
}

// dominantErrorClass returns the most frequent error class, ties broken
// lexically, defaulting to "unknown".
func dominantErrorClass(group []Record) string {
	counts := map[string]int{}
	for _, record := range group {
		for _, step := range record.Set.Steps {
			if step.ErrorClass != "" {
				counts[step.ErrorClass]++
			}
		}
	}
	return dominantKey(counts, "unknown")
}

func dominantKey(counts map[string]int, fallback string) string {
	best := ""
	bestCount := 0
	for key, count := range counts {
		if count > bestCount || (count == bestCount && (best == "" || key < best)) {
			best, bestCount = key, count
		}
	}
	if best == "" {
		return fallback
	}
	return best
}

func dimensionValue(a joblearn.Attribution, dimension attribution.Dimension) string {
	switch dimension {
	case attribution.ByRole:
		return a.Role
	case attribution.ByTrade:
		return a.Trade
	case attribution.ByWorkPackage:
		return a.WorkPackageID
	default:
		return ""
	}
}

func evidenceOf(group []Record) []joblearn.Reference {
	seen := map[string]bool{}
	var out []joblearn.Reference
	for _, record := range group {
		for _, ref := range record.Result.Attribution.Evidence {
			key := ref.Kind + "\x00" + ref.ID + "\x00" + ref.Version
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, ref)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func dedupSort(candidates []joblearn.Candidate) []joblearn.Candidate {
	byKey := map[string]joblearn.Candidate{}
	for _, c := range candidates {
		key := string(c.Kind) + "\x00" + c.Scope + "\x00" + c.Title
		if existing, ok := byKey[key]; ok {
			if c.Confidence > existing.Confidence {
				existing.Confidence = c.Confidence
			}
			existing.Evidence = mergeRefs(existing.Evidence, c.Evidence)
			byKey[key] = existing
			continue
		}
		byKey[key] = c
	}
	out := make([]joblearn.Candidate, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func mergeRefs(a, b []joblearn.Reference) []joblearn.Reference {
	seen := map[string]bool{}
	var out []joblearn.Reference
	for _, ref := range append(append([]joblearn.Reference(nil), a...), b...) {
		key := ref.Kind + "\x00" + ref.ID + "\x00" + ref.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// rate returns successes/outcomes, or 0 for an empty scope.
func rate(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Scope prefixes.
const (
	scopeRole        = "role:"
	scopeTrade       = "trade:"
	scopeWorkPackage = "work_package:"
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
