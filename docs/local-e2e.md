# Local End-to-End Flow

This is the shortest full loop for trying CompatGate on one machine.

## Requirements

- Go 1.26+
- Node.js 22+
- npm 10+

## 1. Start the API

```bash
cp .env.api.example .env.api
export COMPATGATE_API_ADDR=:8080
export COMPATGATE_STORE_DRIVER=sqlite
export COMPATGATE_DB_PATH=./compatgate.db
export COMPATGATE_WEB_ORIGIN=http://localhost:3000
export COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=false
go run ./cmd/compatgate-api
```

PowerShell:

```powershell
$env:COMPATGATE_API_ADDR=":8080"
$env:COMPATGATE_STORE_DRIVER="sqlite"
$env:COMPATGATE_DB_PATH="./compatgate.db"
$env:COMPATGATE_WEB_ORIGIN="http://localhost:3000"
$env:COMPATGATE_ALLOW_REMOTE_HEADER_AUTH="false"
go run ./cmd/compatgate-api
```

## 2. Start the dashboard

```bash
cd apps/web
npm install
npm run dev
```

Open `http://localhost:3000/auth/signin`, create a dev session, then open `http://localhost:3000/projects`.

## 3. Create a project and collect credentials

From the dashboard:

1. Create a new project
2. Copy the `projectId`
3. Copy the `projectToken`

You can also create a project through the CLI:

```bash
go run ./cmd/compatgate project create \
  --user compatgate-dev \
  --name "CompatGate Demo" \
  --repository compatgate/demo \
  --default-protocol openapi \
  --cloud-url http://localhost:8080
```

PowerShell:

```powershell
go run ./cmd/compatgate project create `
  --user compatgate-dev `
  --name "CompatGate Demo" `
  --repository compatgate/demo `
  --default-protocol openapi `
  --cloud-url http://localhost:8080
```

You can also create a project through the API:

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

The response contains both `id` and `projectToken`.

## 4. Run a diff

```bash
go run ./cmd/compatgate diff \
  --protocol openapi \
  --base ./examples/openapi/base.yaml \
  --revision ./examples/openapi/revision.yaml \
  --format json \
  --output ./compatgate-report.json
```

## 5. Upload the report

```bash
go run ./cmd/compatgate upload \
  --input ./compatgate-report.json \
  --cloud-url http://localhost:8080 \
  --project-token "$COMPATGATE_PROJECT_TOKEN" \
  --project-id "$COMPATGATE_PROJECT_ID" \
  --repository compatgate/demo \
  --sha local-demo \
  --ref refs/heads/main
```

PowerShell:

```powershell
go run ./cmd/compatgate upload `
  --input .\compatgate-report.json `
  --cloud-url http://localhost:8080 `
  --project-token $env:COMPATGATE_PROJECT_TOKEN `
  --project-id $env:COMPATGATE_PROJECT_ID `
  --repository compatgate/demo `
  --sha local-demo `
  --ref refs/heads/main
```

## 6. View the uploaded run

Go back to the dashboard:

- `/projects`
- `/projects/<projectId>`
- `/projects/<projectId>/runs/<runId>`

The CLI upload command prints the created `run_id` and `run_url`.
