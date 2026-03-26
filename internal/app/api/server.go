package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/storage"
	"github.com/compatgate/compatgate/internal/storage/memory"
	"github.com/compatgate/compatgate/internal/storage/sqlite"
	"github.com/go-chi/chi/v5"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type responseEnvelope struct {
	Data  any       `json:"data"`
	Error *apiError `json:"error"`
}

type server struct {
	store storage.Store
}

func Run() error {
	addr := envOr("COMPATGATE_API_ADDR", ":8080")
	store, err := buildStore()
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           New(store),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("CompatGate API listening on %s\n", addr)
	return srv.ListenAndServe()
}

func New(store storage.Store) http.Handler {
	srv := &server{store: store}
	router := chi.NewRouter()
	router.Use(jsonHeaders)
	router.Use(cors)
	router.Get("/api/v1/healthz", srv.health)
	router.Route("/api/v1/projects", func(r chi.Router) {
		r.Use(srv.requireUser)
		r.Get("/", srv.listProjects)
		r.Post("/", srv.createProject)
		r.Get("/{projectID}", srv.getProject)
		r.Get("/{projectID}/runs", srv.listRuns)
		r.Get("/{projectID}/runs/{runID}", srv.getRun)
		r.Get("/{projectID}/runs/{runID}/report", srv.getRunReport)
	})
	router.Post("/api/v1/ingest/runs", srv.ingestRun)
	return router
}

func buildStore() (storage.Store, error) {
	driver := strings.ToLower(envOr("COMPATGATE_STORE_DRIVER", "memory"))
	if driver == "sqlite" {
		path := envOr("COMPATGATE_DB_PATH", "compatgate.db")
		return sqlite.New(path)
	}
	return memory.New(), nil
}

func jsonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", envOr("COMPATGATE_WEB_ORIGIN", "http://localhost:3000"))
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CompatGate-User")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := strings.TrimSpace(r.Header.Get("X-CompatGate-User"))
		if user == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		if !allowInsecureHeaderAuth(r.RemoteAddr) {
			writeError(w, http.StatusForbidden, "forbidden", "Header-based auth is limited to local development")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type userContextKey struct{}

func userFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userContextKey{}).(string)
	return value
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Health(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "unhealthy", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects(r.Context(), userFromContext(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_projects_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createProject(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name            string `json:"name"`
		Repository      string `json:"repository"`
		DefaultProtocol string `json:"defaultProtocol"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	project, err := s.store.CreateProject(r.Context(), storage.CreateProjectInput{
		Name:            strings.TrimSpace(payload.Name),
		Repository:      strings.TrimSpace(payload.Repository),
		DefaultProtocol: storageProtocol(payload.DefaultProtocol),
		Owner:           userFromContext(r.Context()),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_project_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *server) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.store.GetProject(r.Context(), userFromContext(r.Context()), chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "project_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *server) listRuns(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListRuns(r.Context(), userFromContext(r.Context()), chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "runs_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), userFromContext(r.Context()), chi.URLParam(r, "projectID"), chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "run_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *server) getRunReport(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), userFromContext(r.Context()), chi.URLParam(r, "projectID"), chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "run_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run.Report)
}

func (s *server) ingestRun(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing_token", "Project token required")
		return
	}
	project, err := s.store.GetProjectByToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}
	var payload struct {
		ProjectID string `json:"projectId"`
		Status    string `json:"status"`
		Protocol  string `json:"protocol"`
		Git       struct {
			Repository string `json:"repository"`
			SHA        string `json:"sha"`
			Ref        string `json:"ref"`
		} `json:"git"`
		Report findings.Report `json:"report"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if payload.ProjectID != "" && payload.ProjectID != project.ID {
		writeError(w, http.StatusForbidden, "project_mismatch", "project token does not match project id")
		return
	}
	run, err := s.store.CreateRun(r.Context(), storage.CreateRunInput{
		ProjectID:  project.ID,
		Status:     defaultString(payload.Status, "completed"),
		Protocol:   storageProtocol(payload.Protocol),
		Repository: payload.Git.Repository,
		SHA:        payload.Git.SHA,
		Ref:        payload.Git.Ref,
		Report:     payload.Report,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_run_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"runId":  run.ID,
		"runUrl": fmt.Sprintf("/projects/%s/runs/%s", project.ID, run.ID),
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseEnvelope{Data: data})
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseEnvelope{Error: &apiError{Code: code, Message: message}})
}

func envOr(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func storageProtocol(value string) findings.Protocol {
	protocol, err := findings.ParseProtocol(value)
	if err != nil {
		return findings.ProtocolOpenAPI
	}
	return protocol
}

func allowInsecureHeaderAuth(remoteAddr string) bool {
	if strings.EqualFold(envOr("COMPATGATE_ALLOW_REMOTE_HEADER_AUTH", "false"), "true") {
		return true
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
