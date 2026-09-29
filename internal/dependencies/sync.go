package dependencies

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Sync rewrites the dependsOn list of every project in the config source to
// match deps, leaving the rest of the document intact.
// Projects absent from deps are left alone.
func Sync(src []byte, deps map[string][]string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("config is empty")
	}

	projectsNode := mapValue(doc.Content[0], "projects")
	if projectsNode == nil || projectsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config has no `projects` mapping")
	}

	for i := 0; i+1 < len(projectsNode.Content); i += 2 {
		name := projectsNode.Content[i].Value
		want, ok := deps[name]
		if !ok {
			continue
		}
		if err := setDependsOn(projectsNode.Content[i+1], want); err != nil {
			return nil, fmt.Errorf("project %q: %w", name, err)
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(DetectIndent(src))
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	return buf.Bytes(), nil
}

func setDependsOn(project *yaml.Node, deps []string) error {
	if project.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping")
	}

	at := keyIndex(project, "dependsOn")

	if len(deps) == 0 {
		if at >= 0 {
			project.Content = append(project.Content[:at], project.Content[at+2:]...)
		}
		return nil
	}

	var old *yaml.Node
	if at >= 0 {
		old = project.Content[at+1]
	}

	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, d := range deps {
		item := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: d}
		if prev := itemNamed(old, d); prev != nil {
			item.HeadComment, item.LineComment, item.FootComment = prev.HeadComment, prev.LineComment, prev.FootComment
		}
		seq.Content = append(seq.Content, item)
	}

	if old != nil {
		seq.HeadComment, seq.LineComment, seq.FootComment = old.HeadComment, old.LineComment, old.FootComment
		project.Content[at+1] = seq
		return nil
	}

	project.Content = append(project.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "dependsOn"},
		seq,
	)
	return nil
}

// itemNamed finds the previous entry for a dependency so a comment explaining
// why it is there survives the rewrite.
func itemNamed(seq *yaml.Node, value string) *yaml.Node {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}
	for _, item := range seq.Content {
		if item.Kind == yaml.ScalarNode && item.Value == value {
			return item
		}
	}
	return nil
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if at := keyIndex(node, key); at >= 0 {
		return node.Content[at+1]
	}
	return nil
}

func keyIndex(node *yaml.Node, key string) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// DetectIndent reads the first indented line's width so a re-encoded config
// keeps the file's existing style instead of the encoder's 4-space default.
func DetectIndent(src []byte) int {
	for line := range strings.SplitSeq(string(src), "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || trimmed == line || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return len(line) - len(trimmed)
	}
	return 2
}
