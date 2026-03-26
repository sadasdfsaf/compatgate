# CompatGate

CompatGate is a GitHub-first API compatibility guardrail for `OpenAPI`, `GraphQL`, `gRPC/Protobuf`, and `AsyncAPI`.

It is designed around one shared findings model so the same result can be shown in:

- the `compatgate` CLI
- GitHub Actions
- the Go upload/API service
- the web dashboard in `apps/web`

CompatGate v0.1 focuses on high-value breaking changes instead of full protocol exhaustiveness.

## Requirements

- Go 1.26+
- Node.js 22+
- npm 10+

## Current scope

### OpenAPI / Swagger

- operation removal
- parameter removal
- optional to required parameter changes
- request field tightening
- response field removal
- enum narrowing
- obvious incompatible type changes

### GraphQL

- type removal
- field removal
- argument removal
- nullable to non-null tightening
- enum value removal
- incompatible field or argument type changes

### gRPC / Protobuf

- service removal
- RPC removal
- message field removal
- field number change or reuse
- requiredness tightening
- obvious incompatible field type changes

### AsyncAPI

- channel removal
- operation removal
- payload field removal
- required payload field tightening
- obvious payload field type changes

## Repository layout

```text
.
|-- action.yml
|-- apps/
|-- cmd/
|   |-- compatgate/
|   `-- compatgate-api/
|-- docs/
|-- examples/
|-- internal/
|-- testdata/
|-- web-contracts/
`-- .github/
    |-- examples/
    `-- workflows/
```

## Five-Minute Flow

1. Start the API with SQLite persistence
2. Start the web dashboard
3. Create a project and copy `projectId` plus `projectToken`
4. Run `compatgate diff` against an example contract
5. Run `compatgate upload`
6. Open the uploaded run in the dashboard

The full walkthrough lives in [docs/local-e2e.md](/E:/CompatGate/docs/local-e2e.md).

## CLI

### Compare two contracts

```bash
compatgate diff \
  --protocol openapi \
  --base ./examples/openapi/base.yaml \
  --revision ./examples/openapi/revision.yaml \
  --format json \
  --output ./compatgate-report.json
```

### Fail on breaking changes

```bash
compatgate breaking \
  --protocol graphql \
  --base ./examples/graphql/base.graphql \
  --revision ./examples/graphql/revision.graphql \
  --fail-on error
```

### Re-render an existing report

```bash
compatgate report \
  --input ./compatgate-report.json \
  --format markdown \
  --output ./compatgate-report.md
```

### Upload a report

```bash
compatgate upload \
  --input ./compatgate-report.json \
  --cloud-url http://localhost:8080 \
  --project-token "$COMPATGATE_PROJECT_TOKEN" \
  --project-id "$COMPATGATE_PROJECT_ID" \
  --repository "$GITHUB_REPOSITORY" \
  --sha "$GITHUB_SHA" \
  --ref "$GITHUB_REF"
```

More runnable demo pairs live in [EXAMPLES.md](/E:/CompatGate/EXAMPLES.md).

### CLI command summary

| Command | Purpose |
| --- | --- |
| `compatgate project create` | Create a project and print its `project_id` plus `project_token` |
| `compatgate project list` | List projects visible to a given user |
| `compatgate diff` | Compare contracts and emit a report |
| `compatgate breaking` | Compare contracts, keep only breaking findings, exit non-zero when any exist |
| `compatgate report` | Re-render an existing JSON report as `json`, `markdown`, `html`, or `text` |
| `compatgate upload` | Upload a JSON report to the CompatGate API |

Common analysis flags:

- `--protocol openapi|graphql|grpc|asyncapi`
- `--base <path-or-url>`
- `--revision <path-or-url>`
- `--config <path>`
- `--format text|json|markdown|html`
- `--output <path>`
- `--fail-on error|warn|never`

## Config file

CompatGate reads `.compatgate.yml` with this shape:

```yaml
severity_threshold: warn
ignore_rules: []
include_paths: []
exclude_paths: []
cloud:
  base_url: http://localhost:8080
  project_token: ""
```

## API service

Run the API locally:

```bash
cp .env.api.example .env.api
go run ./cmd/compatgate-api
```

Supported service environment variables:

- `COMPATGATE_API_ADDR`
- `COMPATGATE_STORE_DRIVER`
- `COMPATGATE_DB_PATH`
- `COMPATGATE_WEB_ORIGIN`
- `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH`

The API currently expects authenticated browser requests to include `X-CompatGate-User`, and upload requests to include `Authorization: Bearer <project-token>`.
For safety, header-based browser auth is limited to loopback/local development unless `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=true` is explicitly set.

The repository includes an API env sample in [.env.api.example](/E:/CompatGate/.env.api.example).

## Web dashboard

The repository now includes a minimal Next.js dashboard in [`apps/web`](/E:/CompatGate/apps/web).

Core routes:

- `/`
- `/auth/signin`
- `/projects`
- `/projects/[projectId]`
- `/projects/[projectId]/runs/[runId]`

Environment variables:

- `NEXT_PUBLIC_COMPATGATE_API_BASE_URL`
- `NEXT_PUBLIC_COMPATGATE_DEV_USER`
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `COMPATGATE_WEB_URL`

Run locally:

```bash
cd apps/web
npm install
npm run dev
```

## First Upload

Before `compatgate upload`, you need a `projectId` and `projectToken`.

The easiest path is:

1. Start the API
2. Start the dashboard
3. Open `/auth/signin`
4. Open `/projects`
5. Create a project
6. Copy the `projectId` and `projectToken`

If you prefer the CLI instead of the dashboard:

```bash
compatgate project create \
  --user compatgate-dev \
  --name "CompatGate Demo" \
  --repository compatgate/demo \
  --default-protocol openapi \
  --cloud-url http://localhost:8080
```

If you prefer raw HTTP instead of the dashboard:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -H "X-CompatGate-User: compatgate-dev" \
  -d '{"name":"CompatGate Demo","repository":"compatgate/demo","defaultProtocol":"openapi"}'
```

The response body contains both fields needed by `compatgate upload`.

## GitHub Action

This repository ships a composite action in [action.yml](/E:/CompatGate/action.yml).

Example:

```yaml
name: CompatGate

on:
  pull_request:

jobs:
  compatgate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: compatgate/compatgate@main
        with:
          protocol: openapi
          base: examples/openapi/base.yaml
          revision: examples/openapi/revision.yaml
          config: .compatgate.yml
          fail-on: error
```

Action inputs:

| Input | Required | Description |
| --- | --- | --- |
| `base` | yes | Base contract file or URL |
| `revision` | yes | Revision contract file or URL |
| `config` | no | Path to `.compatgate.yml` |
| `protocol` | yes | `openapi`, `graphql`, `grpc`, or `asyncapi` |
| `fail-on` | no | `error`, `warn`, or `never` |
| `upload-to-cloud` | no | Set to `"true"` to upload the generated report |
| `project-token` | no | Project token for uploads |
| `project-id` | no | Project id for uploads |
| `cloud-url` | no | CompatGate API base URL |

Action outputs:

| Output | Description |
| --- | --- |
| `report-path` | Absolute path to the JSON report on the runner |
| `markdown-path` | Absolute path to the Markdown report on the runner |

## Development

### Go

```bash
go test ./...
```

### Web

```bash
cd apps/web
npm run build
```

### CI examples

- [Quickstart](/E:/CompatGate/docs/quickstart.md)
- [Local end-to-end flow](/E:/CompatGate/docs/local-e2e.md)
- [OpenAPI workflow example](/E:/CompatGate/.github/examples/compatgate-openapi-check.yml)
- [Cloud upload workflow example](/E:/CompatGate/.github/examples/compatgate-cloud-upload.yml)

## License

Apache-2.0
