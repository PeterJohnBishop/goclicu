package tui

import "super-duper-fortnight/clkup"

type uiState int
type ViewDepth int
type LogMsg string

type InitDataMsg struct {
	User       clkup.User
	Workspaces []clkup.Workspace
}
type PlanLoadedMsg struct {
	TeamID string
	PlanID int
}

type WorkspaceData struct {
	Spaces         []clkup.Space
	FoldersBySpace map[string][]clkup.Folder
	ListsByFolder  map[string][]clkup.List
	ListsBySpace   map[string][]clkup.List
	Tasks          []clkup.Task
	TasksByList    map[string][]clkup.Task
	Performance    clkup.Performance
}

type FanOutCompleteMsg struct {
	TeamID      string
	Performance clkup.Performance
}

type ErrMsg struct{ err error }

type ListItem struct {
	ID       string
	Name     string
	Type     string
	Subtitle string
	Ref      any
}
