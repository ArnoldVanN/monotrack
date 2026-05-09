package printer

type Output struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type BumpOutput struct {
	Output        `json:",inline"`
	Version       string `json:"version"`
	BumpKind      string `json:"bumpKind,omitempty"`
	ChangelogPath string `json:"changelogPath,omitempty"`
}

// TODO: actual printer interface
