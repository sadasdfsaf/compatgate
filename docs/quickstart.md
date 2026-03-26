# Quickstart

This guide matches the current CLI and API contract in the repository.

## Requirements

- Go 1.26+
- Node.js 22+
- npm 10+

## 1. Build the CLI

```bash
go build -o ./bin/compatgate ./cmd/compatgate
```

## 2. Create `.compatgate.yml`

```yaml
severity_threshold: warn
ignore_rules: []
include_paths: []
exclude_paths: []
cloud:
  base_url: http://localhost:8080
  project_token: ""
```

Field notes:

- `severity_threshold` filters findings after analysis
- `ignore_rules` suppresses specific rule ids
- `cloud.base_url` is used by `compatgate upload` when `--cloud-url` is omitted
- `cloud.project_token` is used by `compatgate upload` when `--project-token` is omitted

## 3. Run a local diff

### OpenAPI

```bash
compatgate diff \
  --protocol openapi \
  --base ./examples/openapi/base.yaml \
  --revision ./examples/openapi/revision.yaml \
  --config ./.compatgate.yml \
  --format json \
  --output ./compatgate-report.json
```

### Only fail on breaking changes

```bash
compatgate breaking \
  --protocol graphql \
  --base ./examples/graphql/base.graphql \
  --revision ./examples/graphql/revision.graphql \
  --fail-on error
```

### Re-render the JSON report

```bash
compatgate report \
  --input ./compatgate-report.json \
  --format html \
  --output ./compatgate-report.html
```

## 4. Run the API locally

```bash
go run ./cmd/compatgate-api
```

Useful environment variables:

```bash
export COMPATGATE_API_ADDR=:8080
export COMPATGATE_STORE_DRIVER=sqlite
export COMPATGATE_DB_PATH=./compatgate.db
export COMPATGATE_WEB_ORIGIN=http://localhost:3000
export COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=false
```

PowerShell:

```powershell
$env:COMPATGATE_API_ADDR=":8080"
$env:COMPATGATE_STORE_DRIVER="sqlite"
$env:COMPATGATE_DB_PATH="./compatgate.db"
$env:COMPATGATE_WEB_ORIGIN="http://localhost:3000"
$env:COMPATGATE_ALLOW_REMOTE_HEADER_AUTH="false"
```

API behavior summary:

- browser-style project and run endpoints expect `X-CompatGate-User`
- upload requests expect `Authorization: Bearer <project-token>`
- `POST /api/v1/ingest/runs` accepts a full `findings.Report` payload
- header-based browser auth is restricted to local development unless `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=true`

## 5. Create a project and collect `projectId` / `projectToken`

The upload step needs both values first.

### Option A: create a project in the dashboard

1. Start the API
2. Start the web app
3. Open `/auth/signin`
4. Open `/projects`
5. Create a project
6. Copy the `projectId` and `projectToken`

### Option B: create a project with the CLI

```bash
compatgate project create \
  --user compatgate-dev \
  --name "CompatGate Demo" \
  --repository compatgate/demo \
  --default-protocol openapi \
  --cloud-url http://localhost:8080
```

PowerShell:

```powershell
compatgate project create `
  --user compatgate-dev `
  --name "CompatGate Demo" `
  --repository compatgate/demo `
  --default-protocol openapi `
  --cloud-url http://localhost:8080
```

### Option C: create a project over HTTP

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -H "X-CompatGate-User: compatgate-dev" \
  -d '{"name":"CompatGate Demo","repository":"compatgate/demo","defaultProtocol":"openapi"}'
```

PowerShell:

```powershell
$body = @{
  name = "CompatGate Demo"
  repository = "compatgate/demo"
  defaultProtocol = "openapi"
} | ConvertTo-Json

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/api/v1/projects" `
  -Headers @{ "X-CompatGate-User" = "compatgate-dev" } `
  -ContentType "application/json" `
  -Body $body
```

The response from either option includes both `id` and `projectToken`.

## 6. Upload a report

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

PowerShell:

```powershell
compatgate upload `
  --input .\compatgate-report.json `
  --cloud-url http://localhost:8080 `
  --project-token $env:COMPATGATE_PROJECT_TOKEN `
  --project-id $env:COMPATGATE_PROJECT_ID `
  --repository compatgate/demo `
  --sha local-demo `
  --ref refs/heads/main
```

## 7. Run the web dashboard

Create `apps/web/.env.local` if you want to override the defaults:

```bash
NEXT_PUBLIC_COMPATGATE_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_COMPATGATE_DEV_USER=compatgate-dev
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
COMPATGATE_WEB_URL=http://localhost:3000
```

Then run:

```bash
cd apps/web
npm install
npm run dev
```

Notes:

- the current dashboard works out of the box with the dev-user flow on `/auth/signin`
- GitHub OAuth routes are available when `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, and `COMPATGATE_WEB_URL` are configured

## 8. Use the GitHub Action

Minimal workflow step:

```yaml
- uses: compatgate/compatgate@main
  with:
    protocol: openapi
    base: examples/openapi/base.yaml
    revision: examples/openapi/revision.yaml
    config: .compatgate.yml
    fail-on: error
```

Cloud upload variant:

```yaml
- uses: compatgate/compatgate@main
  with:
    protocol: graphql
    base: examples/graphql/base.graphql
    revision: examples/graphql/revision.graphql
    upload-to-cloud: "true"
    project-token: ${{ secrets.COMPATGATE_PROJECT_TOKEN }}
    project-id: ${{ vars.COMPATGATE_PROJECT_ID }}
    cloud-url: ${{ vars.COMPATGATE_CLOUD_URL }}
```

## 9. Current protocol ids

Use these exact values for `--protocol` and workflow `protocol`:

- `openapi`
- `graphql`
- `grpc`
- `asyncapi`
