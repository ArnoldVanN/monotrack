package projects

type GoProject struct {
	*BaseProject
}

func NewGoProject(name, path string, entry bool, t projectType) Project {
	return &GoProject{
		BaseProject: &BaseProject{
			name:       name,
			path:       path,
			entrypoint: entry,
			typeName:   t,
		},
	}
}

func (p *GoProject) UpdateVersion(newVersion string) error {
	return nil
}
