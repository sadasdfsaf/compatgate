package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/storage"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  repository TEXT,
  default_protocol TEXT,
  owner TEXT NOT NULL,
  project_token TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  status TEXT NOT NULL,
  protocol TEXT NOT NULL,
  created_at TEXT NOT NULL,
  finding_count INTEGER NOT NULL,
  breaking_count INTEGER NOT NULL,
  repository TEXT,
  sha TEXT,
  ref TEXT,
  report_json TEXT NOT NULL,
  FOREIGN KEY(project_id) REFERENCES projects(id)
);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) ListProjects(ctx context.Context, owner string) ([]storage.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, repository, default_protocol, owner, project_token, created_at FROM projects WHERE owner = ? ORDER BY created_at DESC`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := make([]storage.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		latestRun, err := s.latestRun(ctx, project.ID)
		if err == nil {
			project.LatestRun = latestRun
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *Store) CreateProject(ctx context.Context, input storage.CreateProjectInput) (storage.Project, error) {
	project := storage.Project{
		ID:              "proj_" + uuid.NewString(),
		Name:            input.Name,
		Repository:      input.Repository,
		DefaultProtocol: input.DefaultProtocol,
		Owner:           input.Owner,
		ProjectToken:    "cg_" + uuid.NewString(),
		CreatedAt:       time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO projects (id, name, repository, default_protocol, owner, project_token, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		project.ID, project.Name, project.Repository, string(project.DefaultProtocol), project.Owner, project.ProjectToken, project.CreatedAt.Format(time.RFC3339))
	return project, err
}

func (s *Store) GetProject(ctx context.Context, owner string, id string) (storage.Project, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, repository, default_protocol, owner, project_token, created_at FROM projects WHERE owner = ? AND id = ?`, owner, id)
	project, err := scanProject(row)
	if err != nil {
		return storage.Project{}, err
	}
	latestRun, err := s.latestRun(ctx, project.ID)
	if err == nil {
		project.LatestRun = latestRun
	}
	return project, nil
}

func (s *Store) GetProjectByToken(ctx context.Context, token string) (storage.Project, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, repository, default_protocol, owner, project_token, created_at FROM projects WHERE project_token = ?`, token)
	return scanProject(row)
}

func (s *Store) ListRuns(ctx context.Context, owner string, projectID string) ([]storage.RunSummary, error) {
	if _, err := s.GetProject(ctx, owner, projectID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, status, protocol, created_at, finding_count, breaking_count, repository, sha, ref FROM runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]storage.RunSummary, 0)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) CreateRun(ctx context.Context, input storage.CreateRunInput) (storage.RunSummary, error) {
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
	reportBytes, err := json.Marshal(input.Report)
	if err != nil {
		return storage.RunSummary{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO runs (id, project_id, status, protocol, created_at, finding_count, breaking_count, repository, sha, ref, report_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.ProjectID, run.Status, string(run.Protocol), run.CreatedAt.Format(time.RFC3339), run.FindingCount, run.BreakingCount, run.Repository, run.SHA, run.Ref, string(reportBytes))
	return run, err
}

func (s *Store) GetRun(ctx context.Context, owner string, projectID string, runID string) (storage.RunDetail, error) {
	if _, err := s.GetProject(ctx, owner, projectID); err != nil {
		return storage.RunDetail{}, err
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, project_id, status, protocol, created_at, finding_count, breaking_count, repository, sha, ref, report_json FROM runs WHERE project_id = ? AND id = ?`, projectID, runID)
	var run storage.RunSummary
	var createdAt string
	var reportJSON string
	if err := row.Scan(&run.ID, &run.ProjectID, &run.Status, &run.Protocol, &createdAt, &run.FindingCount, &run.BreakingCount, &run.Repository, &run.SHA, &run.Ref, &reportJSON); err != nil {
		return storage.RunDetail{}, err
	}
	run.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	var report findings.Report
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		return storage.RunDetail{}, err
	}
	return storage.RunDetail{Run: run, Report: report}, nil
}

func (s *Store) latestRun(ctx context.Context, projectID string) (*storage.RunSummary, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, project_id, status, protocol, created_at, finding_count, breaking_count, repository, sha, ref FROM runs WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`, projectID)
	run, err := scanRun(row)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(row scanner) (storage.Project, error) {
	var project storage.Project
	var createdAt string
	var protocol string
	if err := row.Scan(&project.ID, &project.Name, &project.Repository, &protocol, &project.Owner, &project.ProjectToken, &createdAt); err != nil {
		return storage.Project{}, err
	}
	project.DefaultProtocol = findings.Protocol(protocol)
	project.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return project, nil
}

func scanRun(row scanner) (storage.RunSummary, error) {
	var run storage.RunSummary
	var createdAt string
	if err := row.Scan(&run.ID, &run.ProjectID, &run.Status, &run.Protocol, &createdAt, &run.FindingCount, &run.BreakingCount, &run.Repository, &run.SHA, &run.Ref); err != nil {
		return storage.RunSummary{}, err
	}
	run.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return run, nil
}
