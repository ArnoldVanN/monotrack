package projects

type NodeProject struct {
	*BaseProject
}

func NewNodeProject(name, path string, versioned bool, t projectType) Project {
	return &NodeProject{
		BaseProject: &BaseProject{
			name:      name,
			path:      path,
			versioned: versioned,
			typeName:  t,
		},
	}
}

func (p *NodeProject) UpdateVersion(newVersion string) error {
	return nil
}
