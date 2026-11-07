# Table Mapping

| Legacy | New | Decision |
|---|---|---|
| `sys_user` | `identity_users` | map; password hashes require controlled rehash/reset |
| `sys_user_role` | `access_user_roles` | map after UUID mapping |
| `sys_role` | `access_roles` | map; normalize numeric data scope |
| `sys_role_menu` | `access_role_menus` | map after menu UUID mapping |
| `sys_role_dept` | `access_role_departments` | map after department UUID mapping |
| `sys_menu` | `access_menus`, `access_permissions` | split buttons/API permission codes from navigation |
| `sys_dept` | `organization_departments` | rebuild `ltree` paths |
| `sys_post` | `organization_positions` | map |
| `sys_user_post` | `organization_user_positions` | map |
| `sys_dict_type/data` | `system_dictionary_types/items` | map |
| `sys_config_group` | `system_setting_groups/settings/revisions` | split JSON into typed settings; rotate secrets |
| `sys_login_log` | `audit_login_events` | map with retention policy |
| `sys_oper_log` | `audit_events` | map |
| `sys_api_access_log` | Prometheus/audit_events | summarize; do not import raw unbounded rows |
| `sys_sms_log` | `audit_events` | redact code/content and retain operational metadata |
| `sys_file_group` | `storage_file_groups` | map |
| `sys_file` | `storage_files` | rewrite storage keys and hashes |
| `sys_file_config` | encrypted settings | do not map plaintext credentials |
| `sys_chat_message` | `messaging_messages` | map direct messages |
| `sys_chat_group*` | `messaging_groups/members/messages` | deduplicate members and system messages |
| `sys_notice*` | `notification_notices/recipients/send_logs` | map |
| `sys_user_notice` | `notification_recipients` | map |
| `sys_job/log` | `scheduler_jobs/runs` | remap handlers to safe registered Go handlers |
| `flow_*` | `workflow_*` | convert Warm-Flow document/state, not table-for-table import |
| `gen_table*` | `codegen_generations` | do not import generated metadata as runtime configuration |
| `student`, `customer` | none | sample/business-specific; excluded without target requirements |
| `sys_server`, `coder_banner` | none | security exclusion / content outside backend scope |
| `qrtz_*` | River tables | do not import |
