package repo

import (
	"testing"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func TestCorruptRecordsFailClosed(t *testing.T) {
	cases := []struct {
		name string
		kind string
		id   string
		data string
		get  func(*store.Store) error
	}{
		{
			name: "project",
			kind: "project_record",
			id:   "p_bad",
			data: `{"contractVersion":"2.0","kind":"project_record","id":"p_bad","revision":1,"name":"bad","state":"BOGUS","layoutVersion":"1"}`,
			get: func(st *store.Store) error {
				_, err := NewProjectRepository(st, projection.NewMem()).Get("p_bad")
				return err
			},
		},
		{
			name: "work package state",
			kind: "work_package",
			id:   "wp_bad",
			data: `{"contractVersion":"2.0","kind":"work_package","id":"wp_bad","projectId":"p1","revision":1,"title":"bad","state":"BOGUS","acceptance":["done"]}`,
			get: func(st *store.Store) error {
				_, err := NewWorkPackageRepository(st, projection.NewMem()).Get("wp_bad")
				return err
			},
		},
		{
			name: "assignment state",
			kind: "assignment",
			id:   "asg_bad",
			data: `{"id":"asg_bad","projectId":"p1","workPackageId":"wp_1","workerId":"w1","state":"bogus","attempt":1}`,
			get: func(st *store.Store) error {
				_, err := NewAssignmentRepository(st, projection.NewMem()).Get("asg_bad")
				return err
			},
		},
		{
			name: "approval decision",
			kind: "approval",
			id:   "approval_wp_1",
			data: `{"workPackageId":"wp_1","projectId":"p1","decision":"maybe","actor":"human","decidedAt":"2026-01-01T00:00:00Z"}`,
			get: func(st *store.Store) error {
				_, err := NewApprovalRepository(st, projection.NewMem()).Get("wp_1")
				return err
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, err := store.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			rec, err := store.NewRecord(c.kind, c.id, 1, []byte(c.data))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := st.PutRecord(rec); err != nil {
				t.Fatal(err)
			}
			if err := c.get(st); err == nil {
				t.Fatal("expected corrupt record to fail closed")
			}
		})
	}
}
