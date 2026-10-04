---
name: identity-access
description: "How THIS organization does end-user access. Whenever an app has sign-in, make its experience and permissions role-based — never one flat experience for everyone. Our users belong to standing IdP role groups (surfaced as X-User-Groups); this skill names them and resolves the signed-in caller to their directory person by username (X-User-Name), then scopes by that person's attributes. Apply when adding authentication, when behavior or data must differ by the caller's role, or when you must identify the caller or scope records per user or team. What each role may DO in a given app is that app's own policy, not defined here."
metadata:
  aep:
    kind: custom
---

# Identity & access (our org standard: role-based, directory-backed)

## Sign-in means role-based

When an app has end-user sign-in, its experience and permissions **depend on the
caller's role** — never a single flat experience for everyone. Design for distinct
roles from the start.

## The org's roles

Our users belong to standing **role groups** in the IdP. Each request carries the
caller's groups as **`X-User-Groups`** — the platform's `api-management` contract
validates the token and injects them; resolve the caller's role from there. An app
draws the roles it needs from this catalog and matches them by group name:

- **Compliance Admin** — administers the org's compliance / GRC program.
- **Team Manager** — leads a team in the company directory (that team's manager).
- **Auditor** — the org's audit function (internal or external audit staff).

The catalog is the **org's**, not any one app's — pick the roles the app needs and
ignore the rest. **What each role is allowed to do is the app's own policy** —
derive it from the app's requirements (and refine it with the user); it is not
fixed here.

## Resolve who the caller is

A role says what *kind* of user the caller is; scoping data to *them* (their team,
the records assigned to them) needs their **directory record**. The caller is one
of the directory's people — the directory is an existing org service (see the
[`internal-services`](../internal-services/SKILL.md) skill for consuming it):

- **`X-User-Id`** is an **opaque IdP subject**, not a directory id — a lookup keyed
  on it 404s. Never scope by it or look the caller up with it.
- **`X-User-Name`** is the caller's **username**, the key the directory stores for
  that same person. Resolve the caller to their directory record by matching this
  against the directory's `username` field **exactly** — use the provider's lookup
  (e.g. `?username=<X-User-Name>`); never approximate from a display name or an
  email local-part. Then read whatever the app scopes by (team, directory id, …)
  from that record.

**Done when:** the app presents a distinct experience per role (role read from
`X-User-Groups`), and every per-user / per-team scope is driven by a directory
record resolved from an exact `X-User-Name` match — with no comparison of
`X-User-Id` to a directory id, and no fuzzy matching on name or email.
