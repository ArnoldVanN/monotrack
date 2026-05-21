package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMissingFileReturnsEmpty(t *testing.T) {
	m, err := Read(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Read on missing file should not error, got: %v", err)
	}
	if m.HasPending() {
		t.Errorf("expected empty manifest, got pending: %+v", m.Pending)
	}
}

func TestWriteThenReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".monotrack-manifest.yaml")
	original := &Manifest{
		Pending: []PendingEntry{
			{Project: "api", Tag: "api/v1.2.3", OldVersion: "v1.2.2", NewVersion: "v1.2.3"},
			{Project: "web", Tag: "web/v0.4.0", OldVersion: "v0.3.0", NewVersion: "v0.4.0"},
		},
	}
	if err := Write(path, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got.Pending) != 2 {
		t.Fatalf("pending len = %d, want 2", len(got.Pending))
	}
	// Write sorts by project name; verify the order.
	if got.Pending[0].Project != "api" || got.Pending[1].Project != "web" {
		t.Errorf("pending order = %s,%s want api,web", got.Pending[0].Project, got.Pending[1].Project)
	}
	if got.Pending[0].Tag != "api/v1.2.3" {
		t.Errorf("api tag = %q", got.Pending[0].Tag)
	}
}

func TestWriteSortsPending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.yaml")
	m := &Manifest{
		Pending: []PendingEntry{
			{Project: "zeta", Tag: "zeta/v1.0.0"},
			{Project: "alpha", Tag: "alpha/v1.0.0"},
			{Project: "mid", Tag: "mid/v1.0.0"},
		},
	}
	if err := Write(path, m); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []string{"alpha", "mid", "zeta"}
	for i, w := range want {
		if got.Pending[i].Project != w {
			t.Errorf("pending[%d] = %s, want %s", i, got.Pending[i].Project, w)
		}
	}
}

func TestHasPending(t *testing.T) {
	if (&Manifest{}).HasPending() {
		t.Error("empty manifest should not have pending")
	}
	m := &Manifest{Pending: []PendingEntry{{Project: "x", Tag: "x/v1"}}}
	if !m.HasPending() {
		t.Error("manifest with entries should report pending")
	}
}

func TestReadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("pending:\n  not a list, just garbage: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("expected error on invalid YAML, got nil")
	}
}
