package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/compatgate/compatgate/internal/findings"
)

func JSON(report findings.Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func Markdown(report findings.Report) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("# CompatGate Report\n\n")
	builder.WriteString(fmt.Sprintf("- Base: `%s`\n", report.Meta.Base))
	builder.WriteString(fmt.Sprintf("- Revision: `%s`\n", report.Meta.Revision))
	builder.WriteString(fmt.Sprintf("- Findings: `%d`\n", report.Summary.FindingCount))
	builder.WriteString(fmt.Sprintf("- Breaking: `%d`\n\n", report.Summary.BreakingCount))
	builder.WriteString("| Protocol | Severity | Breaking | Resource | Rule | Message |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, finding := range report.Findings {
		builder.WriteString(fmt.Sprintf("| %s | %s | %t | `%s` | `%s` | %s |\n", finding.Protocol, finding.Severity, finding.Breaking, finding.Resource, finding.RuleID, strings.ReplaceAll(finding.Message, "|", "\\|")))
	}
	return []byte(builder.String()), nil
}

func HTML(report findings.Report) ([]byte, error) {
	const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>CompatGate Report</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; background: #08111f; color: #edf4ff; margin: 0; padding: 32px; }
    .summary { display: grid; gap: 12px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); margin-bottom: 24px; }
    .card, table { background: rgba(12, 24, 44, 0.92); border: 1px solid rgba(151, 176, 214, 0.18); border-radius: 18px; }
    .card { padding: 16px; }
    table { width: 100%; border-collapse: collapse; overflow: hidden; }
    th, td { text-align: left; padding: 12px 14px; border-bottom: 1px solid rgba(151, 176, 214, 0.12); }
    th { color: #9fb7d8; font-size: 12px; text-transform: uppercase; letter-spacing: .08em; }
    .pill { display: inline-block; padding: 4px 10px; border-radius: 999px; background: rgba(28, 78, 216, 0.2); }
  </style>
</head>
<body>
  <h1>CompatGate Report</h1>
  <p><strong>Base:</strong> {{.Meta.Base}}<br /><strong>Revision:</strong> {{.Meta.Revision}}</p>
  <div class="summary">
    <div class="card"><strong>Findings</strong><div>{{.Summary.FindingCount}}</div></div>
    <div class="card"><strong>Breaking</strong><div>{{.Summary.BreakingCount}}</div></div>
    <div class="card"><strong>Errors</strong><div>{{.Summary.ErrorCount}}</div></div>
    <div class="card"><strong>Warnings</strong><div>{{.Summary.WarnCount}}</div></div>
  </div>
  <table>
    <thead>
      <tr><th>Protocol</th><th>Severity</th><th>Breaking</th><th>Resource</th><th>Rule</th><th>Message</th></tr>
    </thead>
    <tbody>
      {{range .Findings}}
      <tr>
        <td><span class="pill">{{.Protocol}}</span></td>
        <td>{{.Severity}}</td>
        <td>{{.Breaking}}</td>
        <td>{{.Resource}}</td>
        <td>{{.RuleID}}</td>
        <td>{{.Message}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>`
	tmpl, err := template.New("report").Parse(page)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, report); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
