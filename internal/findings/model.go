package findings

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Protocol string

const (
	ProtocolOpenAPI  Protocol = "openapi"
	ProtocolGraphQL  Protocol = "graphql"
	ProtocolGRPC     Protocol = "grpc"
	ProtocolAsyncAPI Protocol = "asyncapi"
)

func ParseProtocol(value string) (Protocol, error) {
	switch Protocol(strings.ToLower(strings.TrimSpace(value))) {
	case ProtocolOpenAPI:
		return ProtocolOpenAPI, nil
	case ProtocolGraphQL:
		return ProtocolGraphQL, nil
	case ProtocolGRPC, "protobuf":
		return ProtocolGRPC, nil
	case ProtocolAsyncAPI:
		return ProtocolAsyncAPI, nil
	default:
		return "", fmt.Errorf("unsupported protocol %q", value)
	}
}

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

func (s Severity) Rank() int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarn:
		return 2
	default:
		return 1
	}
}

func ParseSeverity(value string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "never":
		return "", nil
	case string(SeverityInfo):
		return SeverityInfo, nil
	case string(SeverityWarn), "warning":
		return SeverityWarn, nil
	case string(SeverityError):
		return SeverityError, nil
	default:
		return "", fmt.Errorf("unsupported severity %q", value)
	}
}

type SourceLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type Finding struct {
	ID             string            `json:"id,omitempty"`
	Protocol       Protocol          `json:"protocol"`
	RuleID         string            `json:"rule_id"`
	Severity       Severity          `json:"severity"`
	Breaking       bool              `json:"breaking"`
	Resource       string            `json:"resource"`
	Message        string            `json:"message"`
	Before         any               `json:"before,omitempty"`
	After          any               `json:"after,omitempty"`
	SourceLocation *SourceLocation   `json:"source_location,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type Summary struct {
	FindingCount  int `json:"finding_count"`
	BreakingCount int `json:"breaking_count"`
	ErrorCount    int `json:"error_count"`
	WarnCount     int `json:"warn_count"`
	InfoCount     int `json:"info_count"`
}

type Meta struct {
	Base        string            `json:"base"`
	Revision    string            `json:"revision"`
	GeneratedAt time.Time         `json:"generated_at"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Report struct {
	Protocols []Protocol `json:"protocols"`
	Summary   Summary    `json:"summary"`
	Findings  []Finding  `json:"findings"`
	Meta      Meta       `json:"meta"`
}

func NewReport(protocols []Protocol, base string, revision string, items []Finding) Report {
	clean := slices.Clone(items)
	return Report{
		Protocols: protocols,
		Summary:   Summarize(clean),
		Findings:  clean,
		Meta: Meta{
			Base:        base,
			Revision:    revision,
			GeneratedAt: time.Now().UTC(),
		},
	}
}

func Summarize(items []Finding) Summary {
	summary := Summary{FindingCount: len(items)}
	for _, item := range items {
		if item.Breaking {
			summary.BreakingCount++
		}
		switch item.Severity {
		case SeverityError:
			summary.ErrorCount++
		case SeverityWarn:
			summary.WarnCount++
		default:
			summary.InfoCount++
		}
	}
	return summary
}

func FilterByThreshold(items []Finding, threshold Severity) []Finding {
	if threshold == "" {
		return slices.Clone(items)
	}
	filtered := make([]Finding, 0, len(items))
	for _, item := range items {
		if item.Severity.Rank() >= threshold.Rank() {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func BreakingOnly(items []Finding) []Finding {
	filtered := make([]Finding, 0, len(items))
	for _, item := range items {
		if item.Breaking {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
