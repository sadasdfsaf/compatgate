package memory

import (
	"context"
	"errors"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/compatgate/compatgate/internal/storage"
	"github.com/google/uuid"
)

type Store struct {
	mu       sync.RWMutex
	projects map[string]storage.Project
	runs     map[string]storage.RunDetail
}

func New() *Store {
	return &Store{
		projects: map[string]storage.Project{},
		runs:     map[string]storage.RunDetail{},
	}
}

func (s *Store) Health(context.Context) error { return nil }

func (s *Store) ListProjects(_ context.Context, owner string) ([]storage.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]storage.Project, 0)
	for _, project := range s.projects {
		if project.Owner == owner {
			items = append(items, project)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) CreateProject(_ context.Context, input storage.CreateProjectInput) (storage.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	project := storage.Project{
		ID:              "proj_" + uuid.NewString(),
		Name:            input.Name,
		Repository:      input.Repository,
		DefaultProtocol: input.DefaultProtocol,
		Owner:           input.Owner,
		ProjectToken:    "cg_" + uuid.NewString(),
		CreatedAt:       time.Now().UTC(),
	}
	s.projects[project.ID] = project
	return project, nil
}

func (s *Store) GetProject(_ context.Context, owner string, id string) (storage.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[id]
	if !ok || project.Owner != owner {
		return storage.Project{}, errors.New("project not found")
	}
	return project, nil
}

func (s *Store) GetProjectByToken(_ context.Context, token string) (storage.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, project := range s.projects {
		if project.ProjectToken == token {
			return project, nil
		}
	}
	return storage.Project{}, errors.New("project not found")
}

func (s *Store) ListRuns(_ context.Context, owner string, projectID string) ([]storage.RunSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok || project.Owner != owner {
		return nil, errors.New("project not found")
	}
	items := make([]storage.RunSummary, 0)
	for _, run := range s.runs {
		if run.Run.ProjectID == projectID {
			items = append(items, run.Run)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) CreateRun(_ context.Context, input storage.CreateRunInput) (storage.RunSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[input.ProjectID]
	if !ok {
		return storage.RunSummary{}, errors.New("project not found")
	}
	run := storage.RunSummary{
		ID:            "run_" + uuid.NewString(),
		ProjectID:     input.ProjectID,
		Status:        input.Status,
		Protocol:      input.Protocol,
		CreatedAt:     time.Now().UTC(),
		FindingCount:  input.Report.Summary.FindingCount,
		BreakingCount: input.Report.Summary.BreakingCount,
		Repository:    input.Repository,
		SHA:           input.SHA,
		Ref:           input.Ref,
	}
	s.runs[run.ID] = storage.RunDetail{Run: run, Report: input.Report}
	project.LatestRun = &run
	s.projects[project.ID] = project
	return run, nil
}

func (s *Store) GetRun(_ context.Context, owner string, projectID string, runID string) (storage.RunDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok || project.Owner != owner {
		return storage.RunDetail{}, errors.New("project not found")
	}
	run, ok := s.runs[runID]
	if !ok || run.Run.ProjectID != projectID {
		return storage.RunDetail{}, errors.New("run not found")
	}
	detail := run
	detail.Report.Findings = slices.Clone(detail.Report.Findings)
	return detail, nil
}
