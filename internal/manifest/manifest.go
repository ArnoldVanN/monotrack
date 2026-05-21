// Package manifest reads and writes the .monotrack-manifest.yaml file, which
// records pending tags proposed by a release PR.
package manifest

import (
	"fmt"
	"os"
	"sort"

	"go.yaml.in/yaml/v3"
)

type Manifest struct {
	Pending []PendingEntry `yaml:"pending,omitempty"`
}

type PendingEntry struct {
	Project    string `yaml:"project"`
	Tag        string `yaml:"tag"`
	OldVersion string `yaml:"oldVersion,omitempty"`
	NewVersion string `yaml:"newVersion,omitempty"`
}

// Read loads the manifest from path. Returns an empty manifest if the file
// doesn't exist.
func Read(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{}, nil
		}
		return nil, fmt.Errorf("reading manifest %q: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %q: %w", path, err)
	}
	return &m, nil
}

// Write serializes the manifest to path with Pending sorted by project name
// for stable diffs.
func Write(path string, m *Manifest) error {
	out := *m
	if len(out.Pending) > 0 {
		sorted := make([]PendingEntry, len(out.Pending))
		copy(sorted, out.Pending)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Project < sorted[j].Project })
		out.Pending = sorted
	}

	b, err := yaml.Marshal(&out)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("writing manifest %q: %w", path, err)
	}
	return nil
}

// HasPending reports whether the manifest declares pending tags.
func (m *Manifest) HasPending() bool {
	return len(m.Pending) > 0
}
