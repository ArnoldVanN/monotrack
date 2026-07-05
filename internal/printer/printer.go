package printer

type Output struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type BumpOutput struct {
	Output        `json:",inline"`
	Version       string `json:"version"`
	OldVersion    string `json:"oldVersion,omitempty"`
	BumpKind      string `json:"bumpKind,omitempty"`
	ChangelogPath string `json:"changelogPath,omitempty"`
	// Status is "proposed" (release PR opened, no tags yet) or "released"
	// (tags created and pushed).
	Status string `json:"status,omitempty"`
	PRUrl  string `json:"prUrl,omitempty"`
}

// PruneOutput is one tag selected for (or deleted by) a prune run.
type PruneOutput struct {
	Project string `json:"project"`
	Tag     string `json:"tag"`
	Version string `json:"version"`
	// Deleted is true when the tag was actually removed (--apply), false on a
	// dry run.
	Deleted bool `json:"deleted"`
}

// TODO: actual printer interface
