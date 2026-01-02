package projects

type GoProject struct {
	name         string
	path         string
	typeName     projectType
	dependencies []Project
	entrypoint   bool
}

func NewGoProject(name, path string, entry bool) Project {
	return &GoProject{
		name:       name,
		path:       path,
		entrypoint: entry,
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

func (p *GoProject) AddDependency(proj Project) {
	p.dependencies = append(p.dependencies, proj)
}

func (p *GoProject) ListDependencies() []Project {
	return p.dependencies
}

func (p *GoProject) IsEntrypoint() bool {
	return p.entrypoint
}
