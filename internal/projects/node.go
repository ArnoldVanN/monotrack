package projects

type NodeProject struct {
	name         string
	path         string
	typeName     projectType
	dependencies []Project
	entrypoint   bool
}

func NewNodeProject(name, path string, entry bool) Project {
	return &NodeProject{
		name:       name,
		path:       path,
		entrypoint: entry,
	}
}

func (p *NodeProject) Name() string {
	return p.name
}

func (p *NodeProject) Path() string {
	return p.path
}

func (p *NodeProject) GetType() projectType {
	return p.typeName
}

func (p *NodeProject) AddDependency(proj Project) {
	p.dependencies = append(p.dependencies, proj)
}

func (p *NodeProject) ListDependencies() []Project {
	return p.dependencies
}

func (p *NodeProject) IsEntrypoint() bool {
	return p.entrypoint
}
