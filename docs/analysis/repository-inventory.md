# Repository Inventory

## Audit Scope

- Source: `C:\Users\admin\Desktop\admin\mars-admin-master`
- Audited on: 2026-11-07
- Git metadata: not present. `.git` is absent, so `master` and `dev-master` cannot be compared. No branch claim is made.
- File totals observed: 259 Java, 38 XML, 1 SQL, 55 Vue, 24 TypeScript, 109 JavaScript files. The SQL dump is about 27 MB and includes business data.

## Original Modules

| Module | Responsibility | Evidence |
|---|---|---|
| `mars-common` | Result envelope, base entity, exceptions, RSA helper | `Result.java`, `PageResult.java`, `BusinessException.java` |
| `mars-infra/mars-db` | MySQL, MyBatis-Plus and data-scope interceptor | `MybatisPlusConfig.java`, `DataPermissionInterceptor.java` |
| `mars-infra/mars-redis` | Redis integration | module POM and service imports |
| `mars-infra/mars-oss` | Local, MinIO, Aliyun OSS, Tencent COS, RustFS storage | storage implementations |
| `mars-infra/mars-sms` | Console, Aliyun, Tencent, Qiniu SMS | SMS implementations |
| `mars-infra/mars-pay` | WeChat Pay and Alipay | payment implementations |
| `mars-infra/mars-push` | DingTalk, Feishu, WeCom and webhook push | push implementations |
| `mars-infra/mars-social` | WeChat, mini program, Alipay and Apple login adapters | social implementations |
| `mars-infra/mars-wechat` | WeChat MP and mini program APIs | `WechatMpService`, `WechatMiniProgramService` |
| `mars-infra/mars-mail` | SMTP mail | `EmailService` |
| `mars-infra/mars-websocket` | Handshake and WebSocket container configuration | WebSocket infrastructure |
| `mars-core/mars-system` | Users, roles, menus, departments, posts, dictionaries, settings, logs, cache monitoring, customers, server credentials | 27 Mapper interfaces |
| `mars-core/mars-auth` | Password, SMS, social and mini program login strategies | strategy package |
| `mars-core/mars-file` | File metadata and file groups | two Mapper interfaces |
| `mars-core/mars-gen` | MySQL schema introspection and Java/Vue generation | gen module |
| `mars-core/mars-message` | Direct chat, groups, notices and delivery logs | message module |
| `mars-core/mars-workflow` | Warm-Flow integration and approval operations | workflow module |
| `mars-core/mars-biz` | Student sample business | `StudentController` |
| `mars-api/mars-admin-api` | Admin endpoints, message WebSocket, SSH WebSocket | 27 controllers |
| `mars-api/mars-app-api` | App/mini program login and profile | `AppAuthController` |
| `mars-api/mars-web-api` | Browser login and profile | `WebAuthController` |
| `mars-job` | Quartz jobs and execution logs | job module |
| `mars-starter` | Spring Boot application and configuration | starter module |
| `mars-ui` | Vue 3 + Naive UI admin frontend | 35 views |
| `mars-uniapp` | UniApp chat and profile client | 10 pages |

## Source vs README Differences

- README states Redis 7, MySQL 8 and Spring Boot 3.2.2; root POM confirms those historical versions.
- README describes social and push factories, but parts of the provider layer are thin adapters and depend on runtime configuration. Capability is documented here from actual interfaces, not assumed operational credentials.
- The repository contains additional server credential management, customer and student sample features that are not prominent in README.
- The SQL dump contains live-looking credentials and personal/business records. They are treated as compromised test data. No values are copied into the new project.
- The repository has no Git metadata despite the task book requiring branch comparison.

## Legacy Design Risks

1. Authentication persists no explicit server-side session record in the new target model; Sa-Token session state and Redis are the main source.
2. MyBatis data-scope rewriting wraps arbitrary SQL in a derived query and uses reflection to mutate `BoundSql`. Some failure paths return an empty filter, which can fail open.
3. The original WebSocket registry is a process-local static map. Multi-instance delivery and reliable catch-up are absent.
4. Direct chat writes and realtime notification are not designed as one transactional/outbox flow.
5. Workflow tables are tied to Warm-Flow's Java model and string statuses.
6. MySQL IDs are `BIGINT`; sending them to JavaScript as numbers risks precision loss.
7. The SQL dump and default configuration include secrets that must be rotated.
8. Many old permission strings use `sys:*` and expose menu visibility separately from API checks.
9. Legacy controllers return a `Result<T>` envelope and sometimes use `msg` while other clients expect `message`.
10. The UI stores the bearer token in persistent Pinia storage and WebSocket URLs pass tokens in query strings. The new backend supports this only through the compatibility layer; secure HttpOnly browser sessions are the preferred new flow.
