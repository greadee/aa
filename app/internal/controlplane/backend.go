package controlplane

// Project is a project summary.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// WorkPackageView is a work-package summary.
type WorkPackageView struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	State     string   `json:"state,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

// AssignmentView is an assignment summary.
type AssignmentView struct {
	ID            string `json:"id"`
	WorkPackageID string `json:"workPackageId"`
	WorkerID      string `json:"workerId"`
	State         string `json:"state"`
	Attempt       int    `json:"attempt"`
}

// Backend supplies the control plane's resources. It is implemented by the
// composition root (app/), which composes memory and the kernel scheduler; the
// transport never reaches into modules itself.
type Backend interface {
	Projects() []Project
	CreateProject(id, name string) error
	WorkPackages(projectID string) []WorkPackageView
	SubmitWork(projectID, title string, dependsOn []string) (string, error)
	Assignments() []AssignmentView
	Approve(gateID string) error
}
