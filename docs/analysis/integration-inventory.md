# Integration Inventory

| Legacy integration | Actual source interface | New contract | Credential state |
|---|---|---|---|
| Console SMS | `SmsService` | `integrations.SMSProvider` | no credential; explicitly a development provider |
| Aliyun/Tencent/Qiniu SMS | provider classes | `SMSProvider` | external |
| WeChat/Alipay payment | `PayService` | `PaymentProvider` | external |
| DingTalk/Feishu/WeCom/push webhooks | `PushService` | `PushProvider` | external |
| WeChat MP, mini program, Alipay, Apple login | `SocialLoginService` | `SocialLoginProvider` | external |
| Local, MinIO, Aliyun OSS, Tencent COS, RustFS | `FileStorage` | `storage.Backend` | local usable; object stores external |
| SMTP | `EmailService` | `MailProvider` | external |
| SSH server credentials and web terminal | `SysServerService`, `SshWebSocketHandler` | excluded | security exclusion |

All real providers must return `ErrNotConfigured` rather than synthesize success. Console SMS is a test-only adapter and is never selected in production configuration.
