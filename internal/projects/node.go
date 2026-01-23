package projects

type NodeProject struct {
	*BaseProject
}

func NewNodeProject(name, path string, entry bool, t projectType) Project {
	return &NodeProject{
		BaseProject: &BaseProject{
			name:       name,
			path:       path,
			entrypoint: entry,
			typeName:   t,
		},
	}
}

func (p *NodeProject) UpdateVersion(newVersion string) error {
	return nil
}
