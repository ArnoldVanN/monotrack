package projects

type NodeProject struct {
	name         string
	path         string
	typeName     projectType
	dependencies []Project
}

func NewNodeProject(name, path string) Project {
	return &NodeProject{
		name: name,
		path: path,
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

func (p *NodeProject) GetVersion() (string, error) {
	// TODO:
	return "", nil
}

func (p *NodeProject) Bump(kind BumpKind) error {
	return nil
}

func (p *NodeProject) AddDependency(proj Project) {
	p.dependencies = append(p.dependencies, proj)
}

func (p *NodeProject) ListDependencies() []Project {
	return p.dependencies
}
