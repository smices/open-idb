# Deleted User Archive Design

## Context

IdBridge currently models deleted users as rows in `users` with
`lifecycle_status = 'deleted'`. This makes deletion reversible in shape even
though the product does not need restore semantics. It also means deleted users
still participate in constraints and joins, including the current
`UNIQUE (entity_id, username)` constraint on `users`.

This caused Feishu login and sync risk: a deleted user can continue to occupy a
username and block a new or existing active account from using it.

The product decision is:

- Deleted users do not need to be restored to their original account.
- Deleted users only need to be retained as records, similar to audit/history.
- Disabled and deleted are different states. Disabled/locked users remain
  managed accounts. Deleted users are no longer managed accounts.

## Goals

- Remove deleted users from active account identity constraints.
- Preserve enough deleted-user data for operations, compliance, and debugging.
- Keep disabled/locked users in `users` with their bindings and roles intact.
- Make user deletion explicit, auditable, and irreversible.
- Avoid a broad rewrite of login, sync, RBAC, and session behavior.

## Non-Goals

- No account restore flow.
- No deleted-user login, sync, role assignment, or binding management.
- No Cloud Run log integration.
- No automatic merge of historical deleted records back into active users.

## Recommended Approach

Create a separate archive table and move deleted users out of active account
tables during deletion.

### Tables

Add `archived_users`:

- `id`: new archive row id.
- `entity_id`: owning entity.
- `original_user_id`: original `users.id`.
- `username`, `display_name`, `email`, `phone`, `avatar_url`,
  `employee_no`, `job_title`, `user_type`, `locale`.
- `primary_source_id`.
- `original_created_at`, `original_updated_at`.
- `archived_at`.
- `archived_by_user_id`: admin/user who initiated deletion when available.
- `archive_reason`: short text reason, default empty.
- `user_snapshot`: JSONB copy of the full user row.
- `bindings_snapshot`: JSONB array of account bindings removed with the user.
- `roles_snapshot`: JSONB array of role assignments removed with the user.

Recommended constraints:

- `UNIQUE (entity_id, original_user_id)` to avoid duplicate archives.
- Index `(entity_id, archived_at DESC)`.
- Index `(entity_id, username)`.

No uniqueness constraint should prevent multiple archived users from having the
same username over time. Archived rows are historical records.

### Active User Model

Keep `users.lifecycle_status` values for active account management:

- `active`
- `disabled`
- `locked`

Stop using `deleted` for active user records. Existing code that filters out
`deleted` can remain temporarily during migration, but new deletion behavior
should remove rows from `users` after archiving.

### Delete Flow

User deletion becomes a transaction:

1. Load the `users` row with `entity_id` and `id`.
2. Load related `account_bindings`.
3. Load related role assignments.
4. Insert one row into `archived_users` with snapshots.
5. Delete dependent active records:
   - sessions for the user
   - account bindings
   - role assignments
   - any other rows that prevent user deletion
6. Delete the row from `users`.
7. Write an audit event:
   - action: existing user disabled/deleted action or new `user.archived`
   - resource_type: `archived_user`
   - resource_id: archive id or original user id
   - before: active user snapshot
   - after: archive metadata

If any step fails, the transaction rolls back and the active account remains
unchanged.

### Directory Sync Behavior

Full sync currently marks missing directory users as `directory_users.status =
'deleted'` and then marks managed users as `users.lifecycle_status = 'deleted'`.

Change the managed-user step to archive-and-delete active users instead of
setting lifecycle status to `deleted`.

Directory users may remain in `directory_users` with `status = 'deleted'`
because they are source directory snapshots, not active login accounts. They do
not occupy `users.username`.

### Login Behavior

Login should only operate on active rows in `users`.

For a Feishu login:

- If an account binding exists, it points to an active `users` row.
- If no account binding exists and auto-provisioning is enabled, a new user may
  be created with a username previously used by an archived user.
- Archived users should not be considered candidates for binding resolution.

### Admin UI

Initial implementation can avoid a full UI for archive browsing.

Minimum admin capability:

- Existing audit log should show the delete/archive event and snapshots.
- User list should not show archived users as active users.

Follow-up UI, if needed:

- Add an "Archived users" read-only page.
- Support search by original username, original user id, and archived date.
- Show snapshots and copy JSON.

### Migration Plan

Migration must be explicit and reversible at the database migration level only
through backups, not through product restore.

1. Add `archived_users`.
2. Backfill existing `users.lifecycle_status = 'deleted'` rows:
   - insert archived snapshots into `archived_users`
   - delete dependent active rows
   - delete the users rows
3. Remove `deleted` from future application behavior.
4. Update generated queries.
5. Keep the existing `users_entity_id_username_key` unique constraint after
   backfill; archived users no longer live in `users`, so the constraint is
   valid for active accounts.

Before backfill, migration should check for dependent tables that reference
`users`. Each dependency needs either deletion, snapshotting, or an explicit
decision to retain by audit only.

### Error Handling

- Duplicate archive row for the same `(entity_id, original_user_id)` should be
  treated as idempotent only if the active user row is already gone.
- If archive insert succeeds but active delete fails, the transaction must roll
  back.
- Login should not see partially archived users because all changes are in one
  transaction.

### Tests

Backend tests:

- Deleting a user writes an archive row and removes the active user.
- Deleting a user releases `users(entity_id, username)` so a new user can use
  the same username.
- Disabled users remain in `users` and keep username uniqueness.
- Full sync archives missing managed users instead of setting
  `lifecycle_status = 'deleted'`.
- Feishu login can auto-create a user whose username exists only in
  `archived_users`.
- Audit event contains enough snapshot data to diagnose deletion.

Migration tests:

- Existing deleted users are backfilled into `archived_users`.
- Existing deleted users no longer block username reuse.
- Active, disabled, and locked users are untouched.

## Open Implementation Notes

- The exact dependent tables must be enumerated from schema before writing the
  migration. Known likely dependencies include sessions, account bindings, role
  assignments, and audit logs.
- Audit logs should not be rewritten. They are immutable event records.
- Archived rows should not foreign-key `original_user_id` back to `users`
  because the active row is intentionally removed.
