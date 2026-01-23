package printer

type Output struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type BumpOutput struct {
	Output  `json:",inline"`
	Version string `json:"version"`
	// OldVersion string `json:"oldVersion"`
	// NewVersion string `json:"newVersion"`
}

// TODO: actual printer interface
