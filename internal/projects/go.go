package projects

import (
	"strings"

	"github.com/arnoldvann/monotrack/internal/git"
)

type GoProject struct {
	name         string
	path         string
	typeName     projectType
	dependencies []Project
}

func NewGoProject(name, path string) Project {
	return &GoProject{
		name: name,
		path: path,
	}
}

func (p *GoProject) Name() string {
	return p.name
}

func (p *GoProject) Path() string {
	return p.path
}

func (p *GoProject) GetType() projectType {
	return p.typeName
}

func (p *GoProject) GetVersion() (string, error) {
	// TODO: read go.mod
	return "", nil
}

func (p *GoProject) GetTags() ([]string, error) {
	t, err := git.GetTags()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(t), "\n")

	tags := make([]string, 0)
	for _, tag := range lines {
		if strings.Contains(tag, p.name) {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (p *GoProject) Bump(kind BumpKind) error {
	return nil
}

func (p *GoProject) AddDependency(proj Project) {
	p.dependencies = append(p.dependencies, proj)
}

func (p *GoProject) ListDependencies() []Project {
	return p.dependencies
}
