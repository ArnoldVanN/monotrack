package projects

type GoProject struct {
	*BaseProject
}

func NewGoProject(name, path string, versioned bool, t projectType) Project {
	return &GoProject{
		BaseProject: &BaseProject{
			name:      name,
			path:      path,
			versioned: versioned,
			typeName:  t,
		},
	}
}

func (p *GoProject) UpdateVersion(newVersion string) error {
	return nil
}
