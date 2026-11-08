# 14 Phase 2.1 Final Acceptance

## Acceptance Matrix

| Area                                       | Result  | Evidence                                                            |
| ------------------------------------------ | ------- | ------------------------------------------------------------------- |
| API target unification                     | PASS    | resolver tests, proxy/server-auth tests, production build           |
| Missing or invalid API configuration       | PASS    | fail-closed unit tests                                              |
| Browser and Server Auth target consistency | PASS    | shared resolver and same-origin proxy                               |
| Access-session refresh                     | PASS    | real Go API, 10-second access TTL, BFF refresh E2E                  |
| Concurrent refresh                         | PASS    | single-flight client unit tests                                     |
| Refresh retry limit                        | PASS    | one original-request retry and no recursive refresh                 |
| 403/5xx/network distinction                | PASS    | client unit tests                                                   |
| Logout                                     | PASS    | real API, confirmation, Cookie cleanup and protected-route redirect |
| CSRF lifetime                              | PASS    | Go Cookie test and refresh integration                              |
| Token storage                              | PASS    | HttpOnly Cookies only; no token in localStorage                     |
| Vue theme values/default                   | PASS    | `light/dark/auto`, default `light`                                  |
| Vue density values/default                 | PASS    | `comfy/regular/compact`, default `regular`, CSS variable E2E        |
| Sidebar persistence                        | PASS    | collapse, reload and SSR attribute E2E                              |
| Hydration and first paint                  | PASS    | Cookie SSR mirror and no hydration error in build/E2E               |
| Recent pages                               | PASS    | unit tests and permission-filtered local history                    |
| Page search and keyboard                   | PASS    | Playwright keyboard navigation and Escape                           |
| User search                                | PASS    | real `user:read` API, super-admin flag, Playwright                  |
| Password change                            | PASS    | real temporary-user lifecycle                                       |
| Logout confirmation                        | PASS    | Playwright Dialog and real logout                                   |
| Local system help                          | PASS    | local rules only; no fake external AI                               |
| Login visual comparison                    | PASS    | Vue/React desktop and mobile light/dark                             |
| Shell visual evidence                      | PASS    | 6 paired desktop scenarios and metrics                              |
| Dashboard visual evidence                  | PASS    | paired mobile light/dark screenshots and metrics                    |
| Responsive matrix                          | PASS    | `390x844`, `768x1024`, `1024x768`, `1440x900`, `1920x1080`          |
| Unit and component tests                   | PASS    | 17 files / 50 tests                                                 |
| Go tests                                   | PASS    | identity HTTP and platform HTTP packages                            |
| OpenAPI types                              | PASS    | generation check and TypeScript                                     |
| Production build                           | PASS    | Next.js production build                                            |
| GitHub Actions remote run                  | NOT RUN | local changes had not yet been pushed when this matrix was produced |

## Conditions

1. GitHub Actions must run against the pushed commit before remote CI can be marked PASS.
2. Vue business pages remain Phase 3 placeholders.
3. Visual pixel differences caused by real canonical data versus the Vue fixture are documented in
   the visual report and are not hidden by masks or reduced thresholds.
4. The Vue reference must remain available locally for visual reproduction.

## Conclusion

`ACCEPTED WITH CONDITIONS`

All locally executable Phase 2.1 requirements passed. The remaining acceptance condition is the
remote GitHub Actions run for the final pushed commit.
