# Relationship Mapping

## Identity And Access

- user -> role: `access_user_roles`
- role -> permission: `access_role_permissions`
- role -> menu: `access_role_menus`
- role -> custom department scope: `access_role_departments`
- super administrator: explicit `access_roles.is_super_admin`

## Organization

- user -> department: `organization_memberships`
- one primary department per user: partial unique index
- user -> position: `organization_user_positions`
- department parent: self foreign key plus `ltree` path

## Messaging

- direct message: sender and receiver, no group
- group message: sender and group, no receiver
- group membership: `messaging_group_members`
- read receipt: message/user composite key

## Notification

- notice -> user: `notification_recipients`
- notice -> delivery channel: `notification_send_logs`

## Workflow

- definition -> versioned JSON graph
- instance -> definition
- task -> instance and optional assignee
- event -> instance and optional task/actor

Imports must validate orphan rows and relationship cardinality before enabling constraints.
