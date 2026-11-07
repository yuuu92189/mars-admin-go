# Legacy Database Inventory

The audited dump contains 41 physical tables: 30 business tables and 11 Quartz tables. Stored business rows are not migrated as part of the initial schema; a separate import command produces a validated report.

## Identity And Access

| Legacy table | New tables |
|---|---|
| `sys_user` | `identity_users` |
| `sys_user_role` | `access_user_roles` |
| `sys_role` | `access_roles` |
| `sys_role_menu` | `access_role_menus` |
| `sys_role_dept` | `access_role_departments` |
| `sys_menu` | `access_menus`, `access_permissions` |
| `sys_user_blacklist` | direct-chat block relation added through messaging policy |

## Organization And System

| Legacy table | New tables |
|---|---|
| `sys_dept` | `organization_departments` (`ltree` path instead of `ancestors` CSV) |
| `sys_post` | `organization_positions` |
| `sys_user_post` | `organization_user_positions` |
| `sys_dict_type` | `system_dictionary_types` |
| `sys_dict_data` | `system_dictionary_items` |
| `sys_config_group` | `system_setting_groups`, `system_settings`, `system_setting_revisions` |

## Audit And Monitoring

| Legacy table | New tables |
|---|---|
| `sys_login_log` | `audit_login_events` |
| `sys_oper_log` | `audit_events` |
| `sys_api_access_log` | Prometheus metrics and `audit_events` |
| `sys_server` | not migrated; SSH credential storage and remote execution are excluded |
| `sys_sms_log` | `audit_events` plus provider delivery metadata |

## Files, Messaging, Jobs And Workflow

| Legacy table | New tables |
|---|---|
| `sys_file_group` | `storage_file_groups` |
| `sys_file` | `storage_files` |
| `sys_file_config` | encrypted `system_settings` |
| `sys_chat_message` | `messaging_messages` |
| `sys_chat_group` | `messaging_groups` |
| `sys_chat_group_member` | `messaging_group_members` |
| `sys_chat_group_message` | `messaging_messages` |
| `sys_notice` | `notification_notices` |
| `sys_user_notice` | `notification_recipients` |
| `sys_notice_send_log` | `notification_send_logs` |
| `sys_job`, `sys_job_log` | `scheduler_jobs`, `scheduler_job_runs`, River tables |
| `flow_*` | `workflow_definitions`, instances, tasks, events |
| `gen_table`, `gen_table_column` | live PostgreSQL introspection and `codegen_generations` |
| `student`, `customer` | not migrated as first-class domains |
| `coder_banner` | not mapped; content asset outside backend scope |

## Known Data Quality Issues

- MySQL `BIGINT` identifiers are unsafe as JavaScript numbers.
- Department ancestry is stored as comma-separated text and can be inconsistent.
- Chat read state is embedded on individual messages instead of per-recipient receipts.
- Chat group system messages are encoded as ordinary message records.
- Several tables use `tinyint` flags and overloaded status values.
- Timestamps use `DATETIME` without timezone semantics.
- The dump includes secrets, phone numbers, email addresses, chat content and credential material. A production migration requires a classified, access-controlled source and post-import key rotation.
