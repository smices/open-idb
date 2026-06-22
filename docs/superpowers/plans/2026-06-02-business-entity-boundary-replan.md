# Business Entity Boundary Replan

## Goal

Replan IdBridge around enterprise-group business entities instead of SaaS tenants.

IdBridge is for one enterprise group. Its "multi-tenant" boundary means multiple internal business entities such as headquarters, branches, subsidiaries, factories, brands, or operating units. These entities are isolated by default, but applications may intentionally allow cross-entity access through explicit policy.

This is a new-start direction. No compatibility layer is required for the old SaaS-tenant interpretation.

## Canonical Terms

- Business entity: the default isolation unit inside the enterprise group.
- Entity administrator: administrator for one business entity.
- Enterprise administrator: administrator for the enterprise control plane. This role manages its entity context and also carries global entity/configuration/application-policy capability in the narrowed scope.
- Application: a system that can be entity-owned or globally shared.
- Cross-entity application access: an explicit exception that lets a principal from one entity access an application or resource exposed by another entity.

Avoid these terms in new product text and new code:

- SaaS tenant
- customer tenant
- tenant onboarding
- billing plan
- public tenant registration

## Product Rules

- Entities are isolated by default.
- Entity-scoped data includes users, identity sources, departments, groups, roles, resource scopes, sessions, sync jobs, and audit logs.
- Applications are not automatically isolated to the user's entity.
- Applications can be group-level shared or entity-owned.
- A user from Entity A can access an application owned by Entity B only through explicit assignment or cross-entity policy.
- Cross-entity application access does not grant visibility into Entity B administration data.
- Enterprise administrators manage business entities and global application policy.
- Entity administrators manage only their entity unless granted system-level capability.

## Data Model Direction

Rename the domain model from tenant to entity:

- `tenants` -> `business_entities`
- `tenant_id` -> `entity_id`
- tenant admin -> entity admin
- tenant branding -> entity branding

Application tables should support shared and cross-entity access:

- `applications.owner_entity_id` nullable; `null` means group-level shared application.
- `applications.cross_entity_mode`: `same_entity_only`, `allow_policy`, or `global_shared`.
- `application_assignments.subject_entity_id` stores where the assigned principal belongs.
- `cross_entity_application_policies` controls source entity, target entity, principal, application, and effect.

ULID remains the global ID strategy for every new table and record.

## API Direction

Prefer new entity-oriented API routes:

- `/api/admin/v1/entities`
- `/api/admin/v1/entities/{entity_id}`
- `/api/admin/v1/applications`
- `/api/admin/v1/applications/{application_id}/entity-access`
- `/api/admin/v1/cross-entity-policies`

Existing tenant-shaped routes do not need compatibility support when the implementation is migrated.

Login routes should stay conceptually entity-scoped:

- `/t/{entity}/admin/login` can remain as a short URL shape if desired, but product copy must call it entity or enterprise entity context.
- Application login resolves entity context from the authorization request, not from a global entity picker.
- There is no separate system administrator login. Enterprise administrator login is the management entry and carries the global management capability set.

## Frontend Direction

Visible product wording should use:

- Entity Management
- Business Entity
- Entity Context
- Entity Administrator
- Group-level Application
- Cross-entity Access

The former "Tenant Management" page should become the entity management page. The UI may keep the route temporarily during development, but no product copy should describe IdBridge as SaaS tenant management.

## Implementation Slices

1. Rename product copy and navigation from tenant to entity.
2. Rename backend domain types, handlers, and API contracts from tenant to entity.
3. Replace schema with `business_entities` and `entity_id`; no compatibility migration is needed.
4. Update seed data: initialize the Sweet Night entity, not a SaaS tenant.
5. Update application schema for `owner_entity_id` and `cross_entity_mode`.
6. Add cross-entity application policy tables and admin APIs.
7. Update application authorization checks to enforce:
   - same-entity access by default,
   - explicit allow for cross-entity access,
   - deny precedence.
8. Update audit logs to record source entity, target entity, application, and policy decision for cross-entity access.
9. Update frontend entity management and application access screens.
10. Add tests for default isolation and explicit cross-entity application access.

## Test Requirements

- Entity A user cannot list, read, or mutate Entity B users by default.
- Entity A admin cannot manage Entity B settings by default.
- Entity A user cannot access Entity B-owned application without explicit policy.
- Entity A user can access Entity B-owned application when explicit allow policy exists.
- Deny policy overrides allow policy.
- Group-level shared application still requires assignment or policy.
- Audit logs record cross-entity access decisions.
