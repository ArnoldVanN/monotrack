package printer

type Output struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BumpOutput struct {
	Output  `json:",inline"`
	Version string `json:"version"`
	// OldVersion string `json:"oldVersion"`
	// NewVersion string `json:"newVersion"`
}
