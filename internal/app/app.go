package app

import (
	"github.com/arnoldvann/monotrack/internal/projects"
)

var State *App

type App struct {
	Config   *projects.Config
	Projects map[string]projects.Project
}

func Init(cfg *projects.Config, projects map[string]projects.Project) {
	State = &App{
		Config:   cfg,
		Projects: projects,
	}
}
