package projects

type HelmProject struct {
	*BaseProject
}

func NewHelmProject(name, path string, entry bool, t projectType) Project {
	return &HelmProject{
		BaseProject: &BaseProject{
			name:       name,
			path:       path,
			entrypoint: entry,
			typeName:   t,
		},
	}
}

func (p *HelmProject) UpdateVersion(newVersion string) error {
	return nil
}
