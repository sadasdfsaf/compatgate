package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/compatgate/compatgate/internal/config"
	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/protocols/asyncapi"
	"github.com/compatgate/compatgate/internal/protocols/graphql"
	"github.com/compatgate/compatgate/internal/protocols/openapi"
	"github.com/compatgate/compatgate/internal/protocols/protobuf"
	"github.com/compatgate/compatgate/internal/report"
	"github.com/compatgate/compatgate/internal/upload"
	"github.com/compatgate/compatgate/internal/version"
	"github.com/compatgate/compatgate/pkg/compatgate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type analysisOptions struct {
	Protocol string
	Base     string
	Revision string
	Config   string
	Format   string
	Output   string
	FailOn   string
}

type uploadOptions struct {
	Input     string
	Config    string
	CloudURL  string
	Token     string
	ProjectID string
	Status    string
	Repo      string
	SHA       string
	Ref       string
}

type projectOptions struct {
	Config          string
	CloudURL        string
	User            string
	Name            string
	Repository      string
	DefaultProtocol string
	Format          string
}

func Execute() error {
	root := &cobra.Command{
		Use:           "compatgate",
		Short:         "CompatGate catches API breaking changes before merge",
		Version:       fmt.Sprintf("%s (%s)", version.Version, version.Commit),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(diffCommand(), breakingCommand(), reportCommand(), uploadCommand(), projectCommand())
	return root.Execute()
}

func diffCommand() *cobra.Command {
	opts := &analysisOptions{Format: "text", Config: ".compatgate.yml", FailOn: "never"}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare a base and revision contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, cfg, err := analyze(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if threshold, err := cfg.Threshold(); err == nil {
				rep.Findings = findings.FilterByThreshold(rep.Findings, threshold)
				rep.Summary = findings.Summarize(rep.Findings)
			}
			if err := emitReport(rep, opts.Format, opts.Output); err != nil {
				return err
			}
			if shouldFail(rep.Findings, opts.FailOn) {
				return errors.New("findings exceeded fail threshold")
			}
			return nil
		},
	}
	addAnalysisFlags(cmd.Flags(), opts)
	return cmd
}

func breakingCommand() *cobra.Command {
	opts := &analysisOptions{Format: "text", Config: ".compatgate.yml"}
	cmd := &cobra.Command{
		Use:   "breaking",
		Short: "Output only breaking changes and exit non-zero when they exist",
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, cfg, err := analyze(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if threshold, err := cfg.Threshold(); err == nil && threshold != "" {
				rep.Findings = findings.FilterByThreshold(rep.Findings, threshold)
			}
			rep.Findings = findings.BreakingOnly(rep.Findings)
			rep.Summary = findings.Summarize(rep.Findings)
			if err := emitReport(rep, opts.Format, opts.Output); err != nil {
				return err
			}
			if len(rep.Findings) > 0 {
				return errors.New("breaking changes detected")
			}
			return nil
		},
	}
	addAnalysisFlags(cmd.Flags(), opts)
	return cmd
}

func reportCommand() *cobra.Command {
	var input string
	var format string
	var output string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render an existing JSON report as markdown, html, json, or text",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return errors.New("--input is required")
			}
			bytes, err := os.ReadFile(input)
			if err != nil {
				return err
			}
			var rep findings.Report
			if err := json.Unmarshal(bytes, &rep); err != nil {
				return err
			}
			return emitReport(rep, format, output)
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "Input JSON report")
	cmd.Flags().StringVar(&format, "format", "markdown", "Output format: text, json, markdown, html")
	cmd.Flags().StringVar(&output, "output", "", "Output file path")
	return cmd
}

func uploadCommand() *cobra.Command {
	opts := &uploadOptions{Config: ".compatgate.yml", Status: "completed"}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a JSON report to the CompatGate API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Input == "" {
				return errors.New("--input is required")
			}
			bytes, err := os.ReadFile(opts.Input)
			if err != nil {
				return err
			}
			var rep findings.Report
			if err := json.Unmarshal(bytes, &rep); err != nil {
				return err
			}
			baseURL := strings.TrimSpace(opts.CloudURL)
			if baseURL == "" {
				cfg, err := config.Load(opts.Config)
				if err != nil {
					return err
				}
				baseURL = cfg.Cloud.BaseURL
				if opts.Token == "" {
					opts.Token = cfg.Cloud.ProjectToken
				}
			}
			protocol, err := reportProtocol(rep)
			if err != nil {
				return err
			}
			client := upload.NewClient(baseURL)
			response, err := client.Upload(cmd.Context(), opts.Token, upload.UploadRequest{
				ProjectID: opts.ProjectID,
				Status:    defaultString(opts.Status, "completed"),
				Protocol:  protocol,
				Git: upload.GitMetadata{
					Repository: opts.Repo,
					SHA:        opts.SHA,
					Ref:        opts.Ref,
				},
				Report: rep,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "run_id=%s\nrun_url=%s\n", response.Data.RunID, response.Data.RunURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Input, "input", "", "Input JSON report")
	cmd.Flags().StringVar(&opts.Config, "config", ".compatgate.yml", "Config file path")
	cmd.Flags().StringVar(&opts.CloudURL, "cloud-url", "", "CompatGate API base URL")
	cmd.Flags().StringVar(&opts.Token, "project-token", "", "Project token")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&opts.Status, "status", "completed", "Run status")
	cmd.Flags().StringVar(&opts.Repo, "repository", "", "Git repository")
	cmd.Flags().StringVar(&opts.SHA, "sha", "", "Git commit SHA")
	cmd.Flags().StringVar(&opts.Ref, "ref", "", "Git ref")
	return cmd
}

func projectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage CompatGate projects in the API service",
	}
	cmd.AddCommand(projectCreateCommand(), projectListCommand())
	return cmd
}

func projectCreateCommand() *cobra.Command {
	opts := &projectOptions{Config: ".compatgate.yml", Format: "text"}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project and print its id and token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.User) == "" || strings.TrimSpace(opts.Name) == "" {
				return errors.New("--user and --name are required")
			}
			client := upload.NewClient(resolveCloudURL(opts.Config, opts.CloudURL))
			response, err := client.CreateProject(cmd.Context(), opts.User, upload.CreateProjectRequest{
				Name:            opts.Name,
				Repository:      opts.Repository,
				DefaultProtocol: opts.DefaultProtocol,
			})
			if err != nil {
				return err
			}
			return emitProject(response.Data, opts.Format)
		},
	}
	addProjectFlags(cmd.Flags(), opts)
	return cmd
}

func projectListCommand() *cobra.Command {
	opts := &projectOptions{Config: ".compatgate.yml", Format: "text"}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects visible to a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.User) == "" {
				return errors.New("--user is required")
			}
			client := upload.NewClient(resolveCloudURL(opts.Config, opts.CloudURL))
			response, err := client.ListProjects(cmd.Context(), opts.User)
			if err != nil {
				return err
			}
			if strings.EqualFold(opts.Format, "json") {
				bytes, err := json.MarshalIndent(response.Data, "", "  ")
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(bytes)
				if err == nil {
					_, _ = cmd.OutOrStdout().Write([]byte("\n"))
				}
				return err
			}
			for _, project := range response.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", project.ID, project.Name, project.DefaultProtocol, project.ProjectToken)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.User, "user", "", "CompatGate user/login")
	cmd.Flags().StringVar(&opts.Config, "config", ".compatgate.yml", "Config file path")
	cmd.Flags().StringVar(&opts.CloudURL, "cloud-url", "", "CompatGate API base URL")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format: text or json")
	return cmd
}

func addAnalysisFlags(flags *pflag.FlagSet, opts *analysisOptions) {
	flags.StringVar(&opts.Protocol, "protocol", "", "Protocol to analyze: openapi, graphql, grpc, asyncapi")
	flags.StringVar(&opts.Base, "base", "", "Base schema or contract path")
	flags.StringVar(&opts.Revision, "revision", "", "Revision schema or contract path")
	flags.StringVar(&opts.Config, "config", ".compatgate.yml", "Config file path")
	flags.StringVar(&opts.Format, "format", "text", "Output format: text, json, markdown, html")
	flags.StringVar(&opts.Output, "output", "", "Output file path")
	flags.StringVar(&opts.FailOn, "fail-on", "never", "Fail threshold: error, warn, or never")
}

func addProjectFlags(flags *pflag.FlagSet, opts *projectOptions) {
	flags.StringVar(&opts.User, "user", "", "CompatGate user/login")
	flags.StringVar(&opts.Name, "name", "", "Project name")
	flags.StringVar(&opts.Repository, "repository", "", "Linked repository")
	flags.StringVar(&opts.DefaultProtocol, "default-protocol", "", "Default protocol")
	flags.StringVar(&opts.Config, "config", ".compatgate.yml", "Config file path")
	flags.StringVar(&opts.CloudURL, "cloud-url", "", "CompatGate API base URL")
	flags.StringVar(&opts.Format, "format", "text", "Output format: text or json")
}

func analyze(ctx context.Context, opts *analysisOptions) (findings.Report, config.Config, error) {
	if opts.Protocol == "" || opts.Base == "" || opts.Revision == "" {
		return findings.Report{}, config.Config{}, errors.New("--protocol, --base, and --revision are required")
	}
	cfg, err := config.Load(opts.Config)
	if err != nil {
		return findings.Report{}, config.Config{}, err
	}
	protocol, err := findings.ParseProtocol(opts.Protocol)
	if err != nil {
		return findings.Report{}, config.Config{}, err
	}
	request := compatgate.Request{Protocol: protocol, Base: opts.Base, Revision: opts.Revision}
	var rep findings.Report
	switch protocol {
	case findings.ProtocolOpenAPI:
		rep, err = openapi.Analyze(ctx, request)
	case findings.ProtocolGraphQL:
		rep, err = graphql.Analyze(ctx, request)
	case findings.ProtocolGRPC:
		rep, err = protobuf.Analyze(ctx, request)
	case findings.ProtocolAsyncAPI:
		rep, err = asyncapi.Analyze(ctx, request)
	default:
		err = fmt.Errorf("unsupported protocol %s", protocol)
	}
	if err != nil {
		return findings.Report{}, config.Config{}, err
	}
	if len(cfg.IgnoreRules) > 0 {
		filtered := make([]findings.Finding, 0, len(rep.Findings))
		for _, item := range rep.Findings {
			if !cfg.ShouldIgnore(item.RuleID) {
				filtered = append(filtered, item)
			}
		}
		rep.Findings = filtered
		rep.Summary = findings.Summarize(filtered)
	}
	return rep, cfg, nil
}

func emitReport(rep findings.Report, format string, output string) error {
	content, err := render(rep, format)
	if err != nil {
		return err
	}
	if output == "" {
		_, err = os.Stdout.Write(content)
		if err == nil && len(content) > 0 && content[len(content)-1] != '\n' {
			_, _ = os.Stdout.WriteString("\n")
		}
		return err
	}
	return os.WriteFile(output, content, 0o644)
}

func render(rep findings.Report, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "", "text", "markdown", "md":
		return report.Markdown(rep)
	case "json":
		return report.JSON(rep)
	case "html":
		return report.HTML(rep)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func emitProject(project upload.Project, format string) error {
	if strings.EqualFold(format, "json") {
		bytes, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(bytes)
		if err == nil {
			_, _ = os.Stdout.WriteString("\n")
		}
		return err
	}
	fmt.Fprintf(os.Stdout, "project_id=%s\nproject_token=%s\nname=%s\nrepository=%s\ndefault_protocol=%s\n", project.ID, project.ProjectToken, project.Name, project.Repository, project.DefaultProtocol)
	return nil
}

func resolveCloudURL(configPath string, cloudURL string) string {
	if strings.TrimSpace(cloudURL) != "" {
		return cloudURL
	}
	cfg, err := config.Load(configPath)
	if err != nil || strings.TrimSpace(cfg.Cloud.BaseURL) == "" {
		return "http://localhost:8080"
	}
	return cfg.Cloud.BaseURL
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func reportProtocol(rep findings.Report) (findings.Protocol, error) {
	if len(rep.Protocols) > 0 {
		return rep.Protocols[0], nil
	}
	if len(rep.Findings) > 0 {
		return rep.Findings[0].Protocol, nil
	}
	return "", errors.New("report does not contain any protocol information")
}

func shouldFail(items []findings.Finding, failOn string) bool {
	threshold, err := findings.ParseSeverity(failOn)
	if err != nil || threshold == "" {
		return false
	}
	for _, item := range items {
		if item.Severity.Rank() >= threshold.Rank() {
			return true
		}
	}
	return false
}
