package graphql

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compatgate/compatgate/pkg/compatgate"
)

func TestAnalyzeDetectsGraphQLBreakingChanges(t *testing.T) {
	root := filepath.Join("..", "..", "..", "testdata", "graphql")
	report, err := Analyze(context.Background(), compatgate.Request{
		Base:     filepath.Join(root, "base.graphql"),
		Revision: filepath.Join(root, "revision.graphql"),
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if report.Summary.BreakingCount < 4 {
		t.Fatalf("expected several breaking findings, got %d", report.Summary.BreakingCount)
	}
	for _, finding := range report.Findings {
		if !strings.Contains(finding.Resource, ":") {
			t.Fatalf("expected unique resource identifier, got %q", finding.Resource)
		}
	}
}
