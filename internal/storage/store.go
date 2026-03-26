package storage

import (
	"context"
	"time"

	"github.com/compatgate/compatgate/internal/findings"
)

type Project struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Repository      string            `json:"repository,omitempty"`
	DefaultProtocol findings.Protocol `json:"defaultProtocol,omitempty"`
	Owner           string            `json:"owner"`
	ProjectToken    string            `json:"projectToken,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	LatestRun       *RunSummary       `json:"latestRun,omitempty"`
}

type RunSummary struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"projectId"`
	Status        string            `json:"status"`
	Protocol      findings.Protocol `json:"protocol"`
	CreatedAt     time.Time         `json:"createdAt"`
	FindingCount  int               `json:"findingCount"`
	BreakingCount int               `json:"breakingCount"`
	Repository    string            `json:"repository,omitempty"`
	SHA           string            `json:"sha,omitempty"`
	Ref           string            `json:"ref,omitempty"`
}

type RunDetail struct {
	Run    RunSummary      `json:"run"`
	Report findings.Report `json:"report"`
}

type CreateProjectInput struct {
	Name            string            `json:"name"`
	Repository      string            `json:"repository,omitempty"`
	DefaultProtocol findings.Protocol `json:"defaultProtocol,omitempty"`
	Owner           string            `json:"owner"`
}

type CreateRunInput struct {
	ProjectID  string
	Status     string
	Protocol   findings.Protocol
	Repository string
	SHA        string
	Ref        string
	Report     findings.Report
}

type Store interface {
	Health(ctx context.Context) error
	ListProjects(ctx context.Context, owner string) ([]Project, error)
	CreateProject(ctx context.Context, input CreateProjectInput) (Project, error)
	GetProject(ctx context.Context, owner string, id string) (Project, error)
	GetProjectByToken(ctx context.Context, token string) (Project, error)
	ListRuns(ctx context.Context, owner string, projectID string) ([]RunSummary, error)
	CreateRun(ctx context.Context, input CreateRunInput) (RunSummary, error)
	GetRun(ctx context.Context, owner string, projectID string, runID string) (RunDetail, error)
}
