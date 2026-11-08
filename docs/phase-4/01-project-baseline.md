# Phase 4 Project Baseline

## Repository

- Repository: `https://github.com/yuuu92189/mars-admin-go`
- Branch: `main`
- Baseline commit: `eb33bb36ab8b841f280784db1f7e9ba321fca3ef`
- Worktree at audit start: clean
- Remote baseline: local `main` matched `origin/main`

## Actual Inventory

| Item | Count | Source |
| --- | ---: | --- |
| Go files | 176 | `rg --files -g '*.go'` |
| Go module directories | 14 | `internal/modules/*` |
| Goose migrations | 25 | `db/migrations/*.sql` |
| Canonical OpenAPI operations | 137 | `TestCanonicalRoutePermissionsMatchOpenAPI` |
| React App Router pages | 18 | `frontend/src/app/**/page.tsx` |
| Go-supported business pages | 13 | Phase 3 route matrix |
| Go test functions | 91 | `rg '^func Test'` |
| Frontend Vitest files | 20 | `rg --files frontend/src -g '*.test.*'` |
| Frontend Vitest tests | 58 | `pnpm test` |
| GitHub Actions workflows | 3 | `.github/workflows/*.yml` |

## Modules

`access`, `audit`, `codegen`, `identity`, `integrations`, `messaging`, `monitoring`,
`notification`, `organization`, `outbox`, `scheduler`, `storage`, `system`, `workflow`.

## Baseline Commands

```text
gofmt -l .
go vet ./...
go test ./...
go test -tags=integration ./test/integration -count=1
pnpm typecheck
pnpm lint
pnpm test
pnpm format:check
pnpm build
```

The Windows host lacks a C compiler, so Race Detector is executed in
`golang:1.27.3-bookworm`. SQLC generation is also run in the Windows shell with
`CGO_ENABLED=0`.
