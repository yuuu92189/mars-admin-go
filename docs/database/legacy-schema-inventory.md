# Legacy Schema Inventory

The audited MySQL dump defines 41 physical tables. 35 contain `INSERT` rows in the provided dump; the generated dry-run report is `docs/migration/import-report.json`.

## Business Tables

Identity/access: `sys_user`, `sys_user_role`, `sys_role`, `sys_role_menu`, `sys_role_dept`, `sys_menu`, `sys_user_blacklist`.

Organization/system: `sys_dept`, `sys_post`, `sys_user_post`, `sys_dict_type`, `sys_dict_data`, `sys_config_group`.

Audit: `sys_login_log`, `sys_oper_log`, `sys_api_access_log`, `sys_sms_log`.

Files/messaging: `sys_file`, `sys_file_group`, `sys_file_config`, `sys_chat_message`, `sys_chat_group`, `sys_chat_group_member`, `sys_chat_group_message`, `sys_notice`, `sys_notice_send_log`, `sys_user_notice`.

Jobs/workflow/codegen: `sys_job`, `sys_job_log`, `flow_definition`, `flow_instance`, `flow_node`, `flow_skip`, `flow_task`, `flow_his_task`, `flow_user`, `gen_table`, `gen_table_column`.

Other observed tables: `student`, `customer`, `sys_server`, `coder_banner`.

## Quartz Tables

`qrtz_blob_triggers`, `qrtz_calendars`, `qrtz_cron_triggers`, `qrtz_fired_triggers`, `qrtz_job_details`, `qrtz_locks`, `qrtz_paused_trigger_grps`, `qrtz_scheduler_state`, `qrtz_simple_triggers`, `qrtz_simprop_triggers`, `qrtz_triggers`.

Quartz tables are not imported. River owns durable task execution in the Go system.

## Audit Limitations

- The directory has no Git metadata, so table differences between `master` and `dev-master` cannot be established.
- The dump includes generated/static-content and operational tables that are not all business domains.
