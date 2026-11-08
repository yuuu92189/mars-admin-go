# 04 Component Mapping

The Vue reference contains 67 components across UI, business, layout, chart and AI groups. Phase 2 intentionally implements the foundation set needed by the shell and login flow; it does not pretend that all 67 are migrated.

## Implemented React Components

| React component | File | Responsibility |
| --- | --- | --- |
| `Button` | `frontend/src/components/ui/button.tsx` | Primary/secondary/ghost/text/danger, loading, icon |
| `IconButton` | `frontend/src/components/ui/button.tsx` | Accessible icon-only actions |
| `TextInput` | `frontend/src/components/ui/input.tsx` | Input, textarea, icon, clear and invalid state |
| `Field` | `frontend/src/components/ui/input.tsx` | Label, required, hint and error |
| `Checkbox` | `frontend/src/components/ui/input.tsx` | Controlled checkbox with keyboard focus |
| `Switch` | `frontend/src/components/ui/input.tsx` | Controlled binary switch |
| `Select` | `frontend/src/components/ui/select.tsx` | Filterable option menu with Escape/outside close |
| `Segmented` | `frontend/src/components/ui/select.tsx` | Theme/density mode control |
| `Card`, `GlassCard` | `frontend/src/components/ui/surface.tsx` | Material surfaces |
| `PageHeader`, `SectionLabel` | `frontend/src/components/ui/surface.tsx` | Page and section hierarchy |
| `Tag`, `Avatar`, `Metric` | `frontend/src/components/ui/surface.tsx` | Status, identity and readouts |
| `EmptyState`, `Skeleton` | `frontend/src/components/ui/surface.tsx` | Honest empty/loading states |
| `Dialog`, `Drawer`, `Dropdown`, `Tooltip` | `frontend/src/components/ui/overlay.tsx` | Portal, Escape, focus and menu behavior |
| `DataTable`, `Pagination`, `Tabs` | `frontend/src/components/ui/data-table.tsx` | Typed data primitives |
| `AppShell` | `frontend/src/components/layout/app-shell.tsx` | Protected layout, responsive shell, palette |
| `Sidebar`, `Header` | `frontend/src/components/layout/sidebar.tsx`, `header.tsx` | Navigation and topbar |
| `AppearanceDialog` | `frontend/src/components/layout/appearance-dialog.tsx` | Theme and density |
| `CommandPalette` | `frontend/src/components/layout/command-palette.tsx` | Menu search and actions |
| `BrandMark` | `frontend/src/components/business/brand-mark.tsx` | Vue hexagonal brand mark and lime status point |
| `LoginBrandPanel` | `frontend/src/features/auth/login-brand-panel.tsx` | Kintsugi copy, four-stage journey and typed terminal |
| `LoginScreen` | `frontend/src/features/auth/login-screen.tsx` | Canonical auth form, captcha, tilt, intro and success motion |

## Deferred Mapping

| Vue component | React target | Phase |
| --- | --- | --- |
| `DataTable`, `ListPage`, `SearchForm` | richer feature table/list components | Phase 3 |
| `PermissionMatrix`, `PermissionTrace`, `PermissionTree` | access feature components | Phase 3 |
| `OrgCanvas`, `DeptTree`, `PostLadder` | organization feature components | Phase 3 |
| `CronBuilder`, `GeneratorPipeline` | scheduler/codegen feature components | Phase 3 |
| `RelationGraph`, `TrendChart`, `HeatGrid`, `Swimlane` | chart components with real data | Phase 3/4 |
| `AiAssistant`, `AskDock` | assistant feature after provider decision | Phase 4+ |

## Rules Preserved

- UI components do not call business APIs.
- Feature views own API calls and state.
- Generated OpenAPI types are consumed rather than hand-maintained DTOs.
- No third-party complete UI framework was introduced.
- Components use explicit TypeScript props and accessible labels.
