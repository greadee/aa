// Package context compiles deterministic, bounded context bundles.
//
// It is pure: callers pass explicit inputs and receive a bundle with a stable
// digest. Retrieval from memory or artifacts happens outside the compiler.
package context

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Input is a candidate context item.
type Input struct {
	Kind string
	ID   string
	Text string
}

// Section is an included item with a token estimate.
type Section struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
}

// Bundle is a compiled context.
type Bundle struct {
	ProjectID     string    `json:"projectId,omitempty"`
	WorkPackageID string    `json:"workPackageId"`
	Sections      []Section `json:"sections"`
	TokenBudget   int       `json:"tokenBudget"`
	TokensUsed    int       `json:"tokensUsed"`
	Truncated     bool      `json:"truncated"`
	Digest        string    `json:"digest"`
}

// EstimateTokens estimates tokens as ceil(runes/4).
func EstimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

// Compiler compiles bounded bundles.
type Compiler struct {
	MaxTokens int
}

// DefaultMaxTokens is used when MaxTokens is not set.
const DefaultMaxTokens = 8000

// Compile returns a deterministic bundle. Inputs are sorted by (kind, id) and
// included until the budget is exhausted; the final section is truncated on a
// rune boundary when it does not fit whole.
func (c Compiler) Compile(projectID, workPackageID string, inputs []Input) Bundle {
	budget := c.MaxTokens
	if budget <= 0 {
		budget = DefaultMaxTokens
	}
	sorted := make([]Input, len(inputs))
	copy(sorted, inputs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		return sorted[i].ID < sorted[j].ID
	})

	bundle := Bundle{ProjectID: projectID, WorkPackageID: workPackageID, TokenBudget: budget}
	used := 0
	for _, in := range sorted {
		if in.Text == "" {
			continue
		}
		tokens := EstimateTokens(in.Text)
		if used+tokens <= budget {
			bundle.Sections = append(bundle.Sections, Section{Kind: in.Kind, ID: in.ID, Text: in.Text, Tokens: tokens})
			used += tokens
			continue
		}
		remaining := budget - used
		if remaining <= 0 {
			bundle.Truncated = true
			break
		}
		truncated := truncateToTokens(in.Text, remaining)
		if truncated == "" {
			bundle.Truncated = true
			break
		}
		bundle.Sections = append(bundle.Sections, Section{Kind: in.Kind, ID: in.ID, Text: truncated, Tokens: EstimateTokens(truncated)})
		used += EstimateTokens(truncated)
		bundle.Truncated = true
		break
	}
	bundle.TokensUsed = used
	bundle.Digest = digest(bundle)
	return bundle
}

func truncateToTokens(text string, maxTokens int) string {
	runes := []rune(text)
	maxRunes := maxTokens * 4
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 1 {
		return ""
	}
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "\u2026"
}

func digest(b Bundle) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%d\n", b.ProjectID, b.WorkPackageID, b.TokenBudget)
	for _, s := range b.Sections {
		fmt.Fprintf(h, "%s\x00%s\x00%d\x00%s\n", s.Kind, s.ID, s.Tokens, s.Text)
	}
	return hex.EncodeToString(h.Sum(nil))
}
