# CompatGate Demo Inputs

This file indexes the ready-to-run demo contracts under `examples/`.

## OpenAPI

- Base: `examples/openapi/base.yaml`
- Revision: `examples/openapi/revision.yaml`
- Highlights:
  - query parameter `role` becomes required
  - enum narrows from `admin/member/guest` to `admin/member`
  - response/request field types tighten

Example:

```bash
compatgate diff --protocol openapi --base ./examples/openapi/base.yaml --revision ./examples/openapi/revision.yaml
```

## GraphQL

- Base: `examples/graphql/base.graphql`
- Revision: `examples/graphql/revision.graphql`
- Highlights:
  - `Query.user(id:)` changes from nullable to required
  - `User.email` disappears
  - `Role.GUEST` disappears

Example:

```bash
compatgate breaking --protocol graphql --base ./examples/graphql/base.graphql --revision ./examples/graphql/revision.graphql
```

## Protobuf

- Base: `examples/protobuf/base.proto`
- Revision: `examples/protobuf/revision.proto`
- Highlights:
  - `Users` service is removed
  - field `id` becomes required
  - field `role` disappears

Example:

```bash
compatgate diff --protocol grpc --base ./examples/protobuf/base.proto --revision ./examples/protobuf/revision.proto
```

## AsyncAPI

- Base: `examples/asyncapi/base.yaml`
- Revision: `examples/asyncapi/revision.yaml`
- Highlights:
  - `users.updated` channel is removed
  - `role` becomes required and is then removed from the payload
  - payload field types tighten (`id`, `profile.nickname`)

Example:

```bash
compatgate diff --protocol asyncapi --base ./examples/asyncapi/base.yaml --revision ./examples/asyncapi/revision.yaml
```

## Test Fixtures

The matching protocol fixtures under `testdata/` use the same base/revision pairs so CLI demos and protocol tests stay aligned.
