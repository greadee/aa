package worker

import (
	"context"
	"errors"
	"testing"
)

func TestFakeScriptedResult(t *testing.T) {
	f := NewFake()
	f.Script("wp_1", Result{Status: "succeeded", Summary: "done", Tests: []TestResult{{Name: "t", Outcome: "passed"}}})
	got, err := f.Run(context.Background(), Request{WorkPackageID: "wp_1", AssignmentID: "asg_1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "succeeded" || len(got.Tests) != 1 {
		t.Fatalf("result = %+v", got)
	}
	if f.CallCount() != 1 || f.Calls()[0].AssignmentID != "asg_1" {
		t.Fatalf("calls = %+v", f.Calls())
	}
}

func TestFakeUnscriptedFails(t *testing.T) {
	f := NewFake()
	got, err := f.Run(context.Background(), Request{WorkPackageID: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.Failure == nil || got.Failure.Class != "no_script" {
		t.Fatalf("result = %+v", got)
	}
}

func TestFakeHonorsContextCancel(t *testing.T) {
	f := NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Run(ctx, Request{WorkPackageID: "wp"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestDisabledBlocks(t *testing.T) {
	got, err := Disabled{}.Run(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "blocked" || got.Failure == nil || got.Failure.Class != "execution_disabled" {
		t.Fatalf("result = %+v", got)
	}
}
