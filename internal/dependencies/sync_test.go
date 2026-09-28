package dependencies

import (
	"strings"
	"testing"
)

const syncSrc = `# top of file
projects:
  api:
    type: go
    path: apps/api
    dependsOn:
      - go-shared # why api needs this
      - gone
  ui:
    type: node
    path: packages/ui
  web:
    type: node
    path: apps/web
`

func TestSyncRewritesAddsAndRemoves(t *testing.T) {
	out, err := Sync([]byte(syncSrc), map[string][]string{
		"api": {"go-shared"},
		"ui":  {},
		"web": {"ui"},
	})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		"# top of file",
		"- go-shared # why api needs this",
		"- ui",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "gone") {
		t.Errorf("stale dependency not removed:\n%s", got)
	}
	if strings.Count(got, "dependsOn") != 2 {
		t.Errorf("want dependsOn on api and web only:\n%s", got)
	}
}

func TestSyncRemovesEmptiedKey(t *testing.T) {
	out, err := Sync([]byte(syncSrc), map[string][]string{"api": {}})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if strings.Contains(string(out), "dependsOn") {
		t.Errorf("dependsOn should be gone:\n%s", out)
	}
}

func TestSyncLeavesUnknownProjectsAlone(t *testing.T) {
	out, err := Sync([]byte(syncSrc), map[string][]string{"web": {"ui"}})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !strings.Contains(string(out), "- gone") {
		t.Errorf("api was not in the detected set and should be untouched:\n%s", out)
	}
}

func TestSyncIsIdempotent(t *testing.T) {
	deps := map[string][]string{"api": {"go-shared"}, "ui": {}, "web": {"ui"}}
	once, err := Sync([]byte(syncSrc), deps)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	twice, err := Sync(once, deps)
	if err != nil {
		t.Fatalf("Sync (second pass): %v", err)
	}
	if string(once) != string(twice) {
		t.Errorf("second pass differs:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestSyncPreservesIndent(t *testing.T) {
	src := "projects:\n    api:\n        type: go\n        path: apps/api\n"
	out, err := Sync([]byte(src), map[string][]string{"api": {"go-shared"}})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !strings.Contains(string(out), "\n    api:\n") {
		t.Errorf("4-space indent not preserved:\n%s", out)
	}
}

func TestSyncRejectsConfigWithoutProjects(t *testing.T) {
	if _, err := Sync([]byte("name: x\n"), map[string][]string{}); err == nil {
		t.Error("expected an error for a config with no projects mapping")
	}
}
