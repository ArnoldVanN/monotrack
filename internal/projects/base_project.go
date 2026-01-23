package projects

type BaseProject struct {
	name         string
	path         string
	typeName     projectType
	dependencies []Project
	entrypoint   bool
}

func (p *BaseProject) Name() string {
	return p.name
}

func (p *BaseProject) Path() string {
	return p.path
}

func (p *BaseProject) GetType() projectType {
	return p.typeName
}

func (p *BaseProject) AddDependency(proj Project) {
	p.dependencies = append(p.dependencies, proj)
}

func (p *BaseProject) ListDependencies() []Project {
	return p.dependencies
}

func (p *BaseProject) IsEntrypoint() bool {
	return p.entrypoint
}
