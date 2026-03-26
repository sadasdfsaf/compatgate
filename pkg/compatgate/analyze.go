package compatgate

import (
	"context"

	"github.com/compatgate/compatgate/internal/findings"
)

type Request struct {
	Protocol findings.Protocol
	Base     string
	Revision string
}

type Analyzer interface {
	Analyze(context.Context, Request) (findings.Report, error)
}
