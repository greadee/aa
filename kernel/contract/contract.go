// Package contract builds immutable, least-privilege execution contracts.
//
// A contract grants the intersection of what a work package requests and what
// the project permits. Anything else is recorded as denied. Missing
// capabilities never grant access; the caller fails closed.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/greadee/aa/registry/capabilities"
)

// Budget bounds an attempt.
type Budget struct {
	MaxCalls           *int     `json:"maxCalls,omitempty"`
	MaxTokens          *int     `json:"maxTokens,omitempty"`
	MaxCostUSD         *float64 `json:"maxCostUsd,omitempty"`
	MaxDurationSeconds *int     `json:"maxDurationSeconds,omitempty"`
}

// RuntimeSpec selects where an attempt runs.
type RuntimeSpec struct {
	Kind     string `json:"kind"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	NodeID   string `json:"nodeId,omitempty"`
}

// Contract is the immutable authority for an attempt.
type Contract struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"projectId,omitempty"`
	WorkPackageID string       `json:"workPackageId"`
	AssignmentID  string       `json:"assignmentId"`
	Capabilities  []string     `json:"capabilities"`
	Denied        []string     `json:"denied"`
	Runtime       *RuntimeSpec `json:"runtime,omitempty"`
	Budget        Budget       `json:"budget"`
	Gates         []string     `json:"gates,omitempty"`
	Digest        string       `json:"digest"`
}

// Request describes the desired contract.
type Request struct {
	ID            string
	ProjectID     string
	WorkPackageID string
	AssignmentID  string
	Requested     []string
	Permitted     []string
	Budget        Budget
	Runtime       *RuntimeSpec
	Gates         []string
}

func sortedUnique(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Build validates and constructs a contract.
func Build(req Request) (Contract, error) {
	if req.WorkPackageID == "" || req.AssignmentID == "" {
		return Contract{}, fmt.Errorf("contract: workPackageId and assignmentId are required")
	}
	for _, cap := range append(append([]string{}, req.Requested...), req.Permitted...) {
		if !capabilities.Known[cap] {
			return Contract{}, fmt.Errorf("contract: unknown capability %q", cap)
		}
	}
	requested := sortedUnique(req.Requested)
	permitted := make(map[string]bool, len(req.Permitted))
	for _, p := range req.Permitted {
		permitted[p] = true
	}
	var granted, denied []string
	for _, cap := range requested {
		if permitted[cap] {
			granted = append(granted, cap)
		} else {
			denied = append(denied, cap)
		}
	}
	c := Contract{
		ID:            req.ID,
		ProjectID:     req.ProjectID,
		WorkPackageID: req.WorkPackageID,
		AssignmentID:  req.AssignmentID,
		Capabilities:  granted,
		Denied:        denied,
		Runtime:       req.Runtime,
		Budget:        req.Budget,
		Gates:         sortedUnique(req.Gates),
	}
	digest, err := c.computeDigest()
	if err != nil {
		return Contract{}, err
	}
	c.Digest = digest
	return c, nil
}

// Has reports whether the contract grants a capability.
func (c Contract) Has(capability string) bool {
	for _, c := range c.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

func (c Contract) computeDigest() (string, error) {
	copy := c
	copy.Digest = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
