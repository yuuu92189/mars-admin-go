# Vue / React Parity Matrix

| Feature            | Vue reference                                                 | React implementation                                                        | API dependency                 | Test evidence                                    | Status              |
| ------------------ | ------------------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------------ | ------------------------------------------------ | ------------------- |
| Login              | Kintsugi brand panel, typed terminal and canonical login form | React/CSS reconstruction with the same geometry and motion language         | `/auth/captcha`, `/auth/login` | Login E2E, login visual comparison               | Aligned             |
| Theme              | `light/dark/auto`, default `light`, `scaffold.appearance`     | Same values, default, storage key and SSR Cookie mirror                     | None                           | Appearance unit tests, Playwright persistence    | Aligned             |
| Density            | `comfy/regular/compact`, default `regular`                    | Same values and real `--row-h` changes                                      | None                           | Appearance tests, CSS variable E2E               | Aligned             |
| Sidebar            | Responsive drawer and persistent collapsed rail               | Same desktop width, mobile drawer, persisted state                          | Menu API                       | Rail persistence and mobile scrim E2E            | Aligned             |
| Command Palette    | Recent pages, search, actions and keyboard interaction        | Same functional groups plus local system help                               | Menus, users                   | Component and Playwright keyboard tests          | Aligned             |
| Recent pages       | Local recent history with limit and filtering                 | Permission-filtered local history, maximum five entries                     | Menu API                       | `recent-pages.test.ts`                           | Aligned             |
| Page search        | Title/path matching and Enter navigation                      | Same behavior with current menu permissions                                 | Menu API                       | Command Palette Playwright                       | Aligned             |
| User search        | Real user search when authorized                              | Real canonical user API, gated by `user:read` or backend super-admin flag   | `/admin/users`                 | Permission tests and real Playwright user search | Aligned             |
| Password change    | Current password, new password, confirm, error and success    | Same Dialog and real API, session revocation and login redirect             | `/auth/password`               | Real temporary-user E2E                          | Aligned             |
| Logout             | Confirmation, real logout and login redirect                  | Same confirmation and API behavior, duplicate submission blocked            | `/auth/logout`                 | Real Playwright logout                           | Aligned             |
| Appearance Dialog  | Theme, density and rail controls                              | Same three themes, three densities and rail behavior                        | None                           | Playwright and visual comparison                 | Aligned             |
| Keyboard and focus | Shortcut, arrows, Enter, Escape and focus restoration         | Same interactions with ARIA combobox/listbox and focus trap                 | None                           | Command Palette component and Playwright tests   | Aligned             |
| AI entry           | Vue has an assistant-style entry                              | No external AI API is fabricated; `?` provides local rule-based system help | None                           | Command Palette tests                            | Explicitly scoped   |
| Dashboard          | Vue trend, activity and governance modules                    | Real canonical counters only; missing modules remain empty                  | `/monitoring/dashboard`        | Dashboard E2E and paired visual evidence         | Partial, Phase 3    |
| Business pages     | Full Vue business screens                                     | Explicit Phase 3 placeholders for known routes, real 404 otherwise          | Menu API                       | Protected-route and 404 E2E                      | Deferred to Phase 3 |

## Notes

- “Aligned” means the user-visible Phase 2.1 behavior and its verification evidence are present.
- “Partial” is not reported as complete. Dashboard visual content remains narrower than Vue because
  the canonical Go API currently exposes counters, not every Vue dashboard module.
- Permission hiding is usability only. Every protected operation is still authorized by the Go API.
