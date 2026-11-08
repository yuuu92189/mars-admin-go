# Phase 4 Frontend Audit

## Structure

- Next.js App Router with React 19 and TypeScript strict mode.
- Same-origin Route Handler proxies `/api/*` to the server-side Go API target.
- Server Components perform cookie-based page access checks; Client Components own interactive
  business state.
- Shared table, overlay, select, toast, command palette, appearance and navigation primitives.

## Findings And Fixes

### P1: multipart uploads were buffered by the BFF

The catch-all Route Handler previously converted non-JSON bodies to `ArrayBuffer`. Large uploads
were therefore materialized in the Next server process before reaching the Go size limit.

Fixed by forwarding `request.body` as a stream with `duplex: "half"`. A Vitest regression test
asserts that the upstream request receives a `ReadableStream`.

### P2: post-login redirect accepted protocol-relative paths

The login screen accepted any path beginning with `/`, including `//host`. It now rejects
protocol-relative and backslash-containing redirects and falls back to `/dashboard`.

### P2: visual test raced the settings-ready state

The login submit button is intentionally disabled until public settings load. The visual test now
waits for the button to be enabled before taking the screenshot.

### P1: login branding presented excluded business domains

The decorative login brand panel still used the Vue reference project's asset management,
vulnerability triage and remediation wording. It was replaced with Identity, Access, Workflow and
Audit stages, commands and status text from the actual Go-supported surface. Login visual
snapshots and viewport baselines were regenerated.

## Remaining Conditions

- Vue/React screenshot fixtures intentionally differ from live database values.
- Historical visual reports are evidence for their recorded commit and viewport only.
- The legacy Vue/UniApp path remains a compatibility surface; new clients should use the React
  cookie/CSRF flow.
