package versioning

import (
	"reflect"
	"sort"
	"testing"

	"github.com/arnoldvann/monotrack/internal/projects"
)

func TestPlanPruneFromTags(t *testing.T) {
	api := projects.NewGoProject("api", "apps/api", true, "go")
	web := projects.NewNodeProject("web", "apps/web", true, "node")
	idle := projects.NewGoProject("idle", "apps/idle", true, "go")

	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{
		"api": {}, "web": {}, "idle": {},
	}}

	tags := map[projects.Project][]string{
		// Latest is a stable release: every rc below it is deletable, all
		// stable releases and the latest are kept.
		api: {
			"api/v0.1.0-rc.1",
			"api/v0.1.0",
			"api/v0.2.0-rc.1",
			"api/v0.2.0-rc.2",
			"api/v0.2.0",
		},
		// Latest tag is itself a prerelease (release in flight): the whole
		// project is left untouched, including its older rc tags.
		web: {
			"web/v1.0.0",
			"web/v1.1.0-rc.1",
			"web/v1.1.0-rc.2",
		},
		// Only a single stable tag: nothing to delete.
		idle: {
			"idle/v0.5.0",
		},
	}

	plan, err := planPruneFromTags(cfg, tags)
	if err != nil {
		t.Fatalf("planPruneFromTags: %v", err)
	}

	gotDelete := plan.Tags()
	sort.Strings(gotDelete)
	wantDelete := []string{"api/v0.1.0-rc.1", "api/v0.2.0-rc.1", "api/v0.2.0-rc.2"}
	if !reflect.DeepEqual(gotDelete, wantDelete) {
		t.Errorf("delete set = %v, want %v", gotDelete, wantDelete)
	}

	wantKept := map[string]string{
		"api":  "api/v0.2.0",
		"web":  "web/v1.1.0-rc.2",
		"idle": "idle/v0.5.0",
	}
	if !reflect.DeepEqual(plan.Kept, wantKept) {
		t.Errorf("kept = %v, want %v", plan.Kept, wantKept)
	}
}

// TestPlanPruneFromTags_RcOrderingDoubleDigits guards the dotted-rc ordering:
// rc.10 must sort above rc.2, so with a stable latest all rc.N are deletable
// and none is mistaken for the latest.
func TestPlanPruneFromTags_RcOrderingDoubleDigits(t *testing.T) {
	svc := projects.NewGoProject("svc", "apps/svc", true, "go")
	// Two projects so the tag scheme stays in multi-project (prefixed) mode.
	cfg := &projects.Config{Projects: map[string]projects.ProjectConfig{"svc": {}, "other": {}}}

	tags := map[projects.Project][]string{
		svc: {
			"svc/v0.1.0-rc.1",
			"svc/v0.1.0-rc.2",
			"svc/v0.1.0-rc.10",
			"svc/v0.1.0",
		},
	}

	plan, err := planPruneFromTags(cfg, tags)
	if err != nil {
		t.Fatalf("planPruneFromTags: %v", err)
	}
	if got, want := len(plan.Delete), 3; got != want {
		t.Fatalf("delete count = %d, want %d (%v)", got, want, plan.Tags())
	}
	if plan.Kept["svc"] != "svc/v0.1.0" {
		t.Errorf("kept latest = %q, want svc/v0.1.0", plan.Kept["svc"])
	}
}
