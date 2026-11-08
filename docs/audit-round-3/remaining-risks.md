# 剩余风险

## RISK-001 外部通知 Provider 不是严格 exactly-once

- 级别：P1
- Outbox 与 River 入队原子。
- River delivery job 有重试和 ByArgs 唯一约束。
- Provider 已成功但本地记录超时，仍可能重复发送。
- 需要 Provider 原生 idempotency key 或业务侧收件人级投递账本。

## RISK-002 Redis Pub/Sub 不是可回放消息流

- 级别：P2
- Redis 发布使用稳定事件 ID 和 Lua `SET NX + PUBLISH`，Outbox 重试不会重复发布。
- Redis Pub/Sub 不保存离线期间的事件。
- WebSocket 断线客户端需要通过历史接口重新同步。

## RISK-003 Outbox payload 含业务内容

- 级别：P2
- 消息、公告和状态事件 payload 存于 PostgreSQL JSONB。
- 这些数据原本已存在于业务表，但 Outbox 增加了另一份短期副本。
- 清理 published 事件后副本删除。
- 不得把 Token、密码、Cookie 等认证秘密写入 Outbox。

## RISK-004 Published 清理早于极端延迟消费

- 级别：P2
- Relay 在 River 入队后即标记 published。
- 默认 7 天清理。
- 如果 River consumer 极端重试超过 7 天，Outbox 记录可能先被清理。
- River 自身 job 保留和监控仍需运维关注。

## RISK-005 文件删除 Provider 变更

- 级别：P2
- 删除事件记录原 StorageProvider。
- Worker 配置与事件 Provider 不一致时 consumer 返回错误并重试。
- 多 Provider 环境需要按 Provider 路由到对应 Backend，而不是只配置一个 Worker Backend。

## RISK-006 Refresh 并发失败请求仍返回 401

- 级别：P2（UX）
- 安全上不会误撤销赢家 Session。
- 输掉竞争的客户端需要重新读取 Cookie 或重新登录。
- 若产品要求所有并发请求返回同一凭证，需要另行实现独立 AEAD 加密的旋转结果缓存。

## RISK-007 Refresh History 无独立归档

- 级别：P2
- 当前采用 retention 后删除。
- 登录审计由 audit 表保留。
- 若合规要求保留 Token Hash 历史，需要新增归档存储和密钥/访问控制。

## RISK-008 字典和系统配置无部门范围

- 级别：P2
- 当前模型是全局配置。
- 继续使用 RBAC。
- 若未来多租户，需要新增 tenant/owner 数据模型。

## RISK-009 Metrics 抓取配置

- 级别：P2（运维）
- `/metrics` 已要求 `audit:read`。
- Prometheus 未配置认证时会收到 401。
- 应使用独立只读监控账号，不应复用管理员长期凭证。

## RISK-010 Outbox CLI 需受控访问

- 级别：P2
- `cmd/outbox` 可列出和重投事件。
- 应以运维身份、受控环境变量和数据库权限运行。
- 不应暴露为普通 HTTP API。

## RISK-011 前端仅完成构建和 API 兼容验证

- 级别：P2
- Vue 生产构建通过。
- 真实 API 集成测试通过。
- 本轮未执行完整浏览器端多标签 Refresh E2E；服务端已覆盖并发场景。

