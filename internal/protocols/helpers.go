package protocols

import (
	"fmt"
	"slices"
	"strings"

	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/normalize"
)

func EnumShrunk(before, after []string) bool {
	if len(after) >= len(before) || len(before) == 0 {
		return false
	}
	afterSet := map[string]bool{}
	for _, value := range after {
		afterSet[value] = true
	}
	for _, value := range before {
		if !afterSet[value] {
			return true
		}
	}
	return false
}

func IncompatibleType(before, after string) bool {
	normalizedBefore := strings.TrimSpace(strings.ToLower(before))
	normalizedAfter := strings.TrimSpace(strings.ToLower(after))
	return normalizedBefore != "" && normalizedAfter != "" && normalizedBefore != normalizedAfter
}

func Finding(
	protocol findings.Protocol,
	ruleID string,
	severity findings.Severity,
	breaking bool,
	resource normalize.Resource,
	message string,
	before any,
	after any,
) findings.Finding {
	return findings.Finding{
		Protocol:       protocol,
		RuleID:         ruleID,
		Severity:       severity,
		Breaking:       breaking,
		Resource:       resource.Identifier,
		Message:        message,
		Before:         before,
		After:          after,
		SourceLocation: resource.Source,
		Labels:         resource.Meta,
	}
}

func ChangedFinding(
	protocol findings.Protocol,
	ruleID string,
	severity findings.Severity,
	breaking bool,
	before normalize.Resource,
	after normalize.Resource,
	message string,
) findings.Finding {
	return findings.Finding{
		Protocol:       protocol,
		RuleID:         ruleID,
		Severity:       severity,
		Breaking:       breaking,
		Resource:       after.Identifier,
		Message:        message,
		Before:         before,
		After:          after,
		SourceLocation: after.Source,
		Labels:         after.Meta,
	}
}

func MergeFindings(parts ...[]findings.Finding) []findings.Finding {
	combined := []findings.Finding{}
	for _, part := range parts {
		combined = append(combined, part...)
	}
	slices.SortStableFunc(combined, func(a, b findings.Finding) int {
		if a.Protocol != b.Protocol {
			return strings.Compare(string(a.Protocol), string(b.Protocol))
		}
		if a.Resource != b.Resource {
			return strings.Compare(a.Resource, b.Resource)
		}
		return strings.Compare(a.RuleID, b.RuleID)
	})
	return combined
}

func Source(path string) *findings.SourceLocation {
	return &findings.SourceLocation{File: path}
}

func AddLabel(meta map[string]string, key, value string) map[string]string {
	if meta == nil {
		meta = map[string]string{}
	}
	meta[key] = value
	return meta
}

func TypeSummary(resource normalize.Resource) string {
	required := "optional"
	if resource.Required {
		required = "required"
	}
	if len(resource.EnumValues) > 0 {
		return fmt.Sprintf("%s %s enum=%s", resource.Kind, required, strings.Join(resource.EnumValues, ","))
	}
	return fmt.Sprintf("%s %s type=%s", resource.Kind, required, resource.Type)
}
