# CompatGate

CompatGate is a GitHub-first API compatibility guardrail for `OpenAPI`, `GraphQL`, `gRPC / Protobuf`, and `AsyncAPI`.

CompatGate 是一个 GitHub-first 的 API 兼容性防护机制，面向 `OpenAPI`、`GraphQL`、`gRPC / Protobuf` 和 `AsyncAPI`。

It is built around one shared findings model so the same compatibility report can flow through:

它围绕一套统一的检查结果模型构建，因此同一份兼容性报告可以贯穿：

- the `compatgate` CLI / `compatgate` CLI
- GitHub Actions / GitHub Actions
- the Go upload / API service / Go 上传与 API 服务
- the web dashboard in `apps/web` / `apps/web` 下的 Web 控制台

CompatGate v0.1 focuses on high-value breaking changes instead of full protocol exhaustiveness.

CompatGate v0.1 先聚焦高价值的破坏性变更，而不是一开始追求协议层面的完全覆盖。

## Requirements / 环境要求

- Go 1.26+
- Node.js 22+
- npm 10+

## Current Scope / 当前范围

### OpenAPI / Swagger

- operation removal / operation 删除
- parameter removal / parameter 删除
- optional to required parameter changes / optional 变 required
- request field tightening / 请求字段收紧
- response field removal / 响应字段删除
- enum narrowing / enum 缩小
- obvious incompatible type changes / 明显的不兼容类型变更

### GraphQL

- type removal / type 删除
- field removal / field 删除
- argument removal / argument 删除
- nullable to non-null tightening / nullable 收紧为 non-null
- enum value removal / enum value 删除
- incompatible field or argument type changes / 字段或参数类型的不兼容变更

### gRPC / Protobuf

- service removal / service 删除
- RPC removal / RPC 删除
- message field removal / message 字段删除
- field number change or reuse / field number 变更或复用
- requiredness tightening / required 性质收紧
- obvious incompatible field type changes / 明显的不兼容字段类型变更

### AsyncAPI

- channel removal / channel 删除
- operation removal / operation 删除
- payload field removal / payload 字段删除
- required payload field tightening / required payload 字段收紧
- obvious payload field type changes / 明显的 payload 字段类型变更

## Repository Layout / 仓库结构

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

## Five-Minute Flow / 五分钟跑通

1. Start the API with SQLite persistence. / 启动带 SQLite 持久化的 API。
2. Start the web dashboard. / 启动 Web 控制台。
3. Create a project and copy `projectId` plus `projectToken`. / 创建项目并复制 `projectId` 和 `projectToken`。
4. Run `compatgate diff` against an example contract. / 对示例契约运行 `compatgate diff`。
5. Run `compatgate upload`. / 执行 `compatgate upload`。
6. Open the uploaded run in the dashboard. / 在控制台中打开上传后的运行记录。

The full walkthrough lives in [docs/local-e2e.md](/E:/CompatGate/docs/local-e2e.md).

完整的端到端流程见 [docs/local-e2e.md](/E:/CompatGate/docs/local-e2e.md)。

## CLI

### Compare Two Contracts / 对比两个契约

```bash
compatgate diff \
  --protocol openapi \
  --base ./examples/openapi/base.yaml \
  --revision ./examples/openapi/revision.yaml \
  --format json \
  --output ./compatgate-report.json
```

### Fail on Breaking Changes / 在破坏性变更时失败

```bash
compatgate breaking \
  --protocol graphql \
  --base ./examples/graphql/base.graphql \
  --revision ./examples/graphql/revision.graphql \
  --fail-on error
```

### Re-render an Existing Report / 重新渲染已有报告

```bash
compatgate report \
  --input ./compatgate-report.json \
  --format markdown \
  --output ./compatgate-report.md
```

### Upload a Report / 上传报告

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

更多可直接运行的示例对在 [EXAMPLES.md](/E:/CompatGate/EXAMPLES.md)。

### CLI Command Summary / CLI 命令总览

| Command | Purpose |
| --- | --- |
| `compatgate project create` | Create a project and print its `project_id` plus `project_token` / 创建项目并输出 `project_id` 和 `project_token` |
| `compatgate project list` | List projects visible to a given user / 列出某个用户可见的项目 |
| `compatgate diff` | Compare contracts and emit a report / 对比契约并输出兼容性报告 |
| `compatgate breaking` | Compare contracts, keep only breaking findings, exit non-zero when any exist / 对比契约，仅保留破坏性变更并在存在时返回非零退出码 |
| `compatgate report` | Re-render an existing JSON report as `json`, `markdown`, `html`, or `text` / 将已有 JSON 报告重新渲染为 `json`、`markdown`、`html` 或 `text` |
| `compatgate upload` | Upload a compatibility report to the CompatGate API / 将兼容性报告上传到 CompatGate API |

Common analysis flags / 常用分析参数：

- `--protocol openapi|graphql|grpc|asyncapi`
- `--base <path-or-url>`
- `--revision <path-or-url>`
- `--config <path>`
- `--format text|json|markdown|html`
- `--output <path>`
- `--fail-on error|warn|never`

## Config File / 配置文件

CompatGate reads `.compatgate.yml` with this shape:

CompatGate 读取 `.compatgate.yml`，格式如下：

```yaml
severity_threshold: warn
ignore_rules: []
include_paths: []
exclude_paths: []
cloud:
  base_url: http://localhost:8080
  project_token: ""
```

## API Service / API 服务

Run the API locally:

在本地运行 API：

```bash
cp .env.api.example .env.api
go run ./cmd/compatgate-api
```

Supported service environment variables / 支持的服务环境变量：

- `COMPATGATE_API_ADDR`
- `COMPATGATE_STORE_DRIVER`
- `COMPATGATE_DB_PATH`
- `COMPATGATE_WEB_ORIGIN`
- `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH`

The API currently expects authenticated browser requests to include `X-CompatGate-User`, and upload requests to include `Authorization: Bearer <project-token>`.

当前 API 默认要求浏览器侧请求带上 `X-CompatGate-User`，上传请求带上 `Authorization: Bearer <project-token>`。

For safety, header-based browser auth is limited to loopback or local development unless `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=true` is explicitly set.

出于安全考虑，基于请求头的浏览器认证默认仅限回环地址或本地开发环境使用，除非显式设置 `COMPATGATE_ALLOW_REMOTE_HEADER_AUTH=true`。

The repository includes an API env sample in [.env.api.example](/E:/CompatGate/.env.api.example).

仓库中提供了 API 环境变量样板：[.env.api.example](/E:/CompatGate/.env.api.example)。

## Web Dashboard / Web 控制台

The repository includes a minimal Next.js dashboard in [`apps/web`](/E:/CompatGate/apps/web).

仓库中包含一个最小可运行的 Next.js Web 控制台，位于 [`apps/web`](/E:/CompatGate/apps/web)。

Core routes / 核心路由：

- `/`
- `/auth/signin`
- `/projects`
- `/projects/[projectId]`
- `/projects/[projectId]/runs/[runId]`

Environment variables / 环境变量：

- `NEXT_PUBLIC_COMPATGATE_API_BASE_URL`
- `NEXT_PUBLIC_COMPATGATE_DEV_USER`
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `COMPATGATE_WEB_URL`

Run locally / 本地运行：

```bash
cd apps/web
npm install
npm run dev
```

## First Upload / 第一次上传

Before `compatgate upload`, you need a `projectId` and `projectToken`.

在执行 `compatgate upload` 前，你需要先拿到 `projectId` 和 `projectToken`。

The easiest path is:

最简单的路径是：

1. Start the API. / 启动 API。
2. Start the dashboard. / 启动控制台。
3. Open `/auth/signin`. / 打开 `/auth/signin`。
4. Open `/projects`. / 打开 `/projects`。
5. Create a project. / 创建项目。
6. Copy the `projectId` and `projectToken`. / 复制 `projectId` 和 `projectToken`。

If you prefer the CLI instead of the dashboard:

如果你更想走 CLI，而不是先打开控制台：

```bash
compatgate project create \
  --user compatgate-dev \
  --name "CompatGate Demo" \
  --repository compatgate/demo \
  --default-protocol openapi \
  --cloud-url http://localhost:8080
```

If you prefer raw HTTP instead of the dashboard:

如果你更想直接走 HTTP：

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -H "X-CompatGate-User: compatgate-dev" \
  -d '{"name":"CompatGate Demo","repository":"compatgate/demo","defaultProtocol":"openapi"}'
```

The response body contains both fields needed by `compatgate upload`.

返回体会包含 `compatgate upload` 所需的两个字段。

## GitHub Action

This repository ships a composite action in [action.yml](/E:/CompatGate/action.yml).

这个仓库提供了一个 composite GitHub Action，位于 [action.yml](/E:/CompatGate/action.yml)。

Example / 示例：

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

Action inputs / Action 输入：

| Input | Required | Description |
| --- | --- | --- |
| `base` | yes | Base contract file or URL / 基准契约文件或 URL |
| `revision` | yes | Revision contract file or URL / 目标契约文件或 URL |
| `config` | no | Path to `.compatgate.yml` / `.compatgate.yml` 路径 |
| `protocol` | yes | `openapi`, `graphql`, `grpc`, or `asyncapi` |
| `fail-on` | no | `error`, `warn`, or `never` |
| `upload-to-cloud` | no | Set to `"true"` to upload the generated report / 设为 `"true"` 时上传生成的报告 |
| `project-token` | no | Project token for uploads / 上传用项目令牌 |
| `project-id` | no | Project id for uploads / 上传用项目 ID |
| `cloud-url` | no | CompatGate API base URL / CompatGate API 地址 |

Action outputs / Action 输出：

| Output | Description |
| --- | --- |
| `report-path` | Absolute path to the JSON report on the runner / runner 上 JSON 报告的绝对路径 |
| `markdown-path` | Absolute path to the Markdown report on the runner / runner 上 Markdown 报告的绝对路径 |

## Development / 开发

### Go

```bash
go test ./...
```

### Web

```bash
cd apps/web
npm run build
```

### CI Examples / CI 示例

- [Quickstart](/E:/CompatGate/docs/quickstart.md)
- [Local end-to-end flow](/E:/CompatGate/docs/local-e2e.md)
- [OpenAPI workflow example](/E:/CompatGate/.github/examples/compatgate-openapi-check.yml)
- [Cloud upload workflow example](/E:/CompatGate/.github/examples/compatgate-cloud-upload.yml)

## License / 许可证

Apache-2.0
