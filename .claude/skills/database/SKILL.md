---
name: database
description: How THIS organization persists an app's own structured data — always a platform-managed PostgreSQL, declared as a platform-resource dependency with resourceType postgres-cnpg. Apply whenever a service must store its own records, rows, entities, relationships, or history that no other internal service already owns (e.g. an app's orders, tickets, comments, or change history). NEVER embed SQLite, a file-based DB, or an in-memory store, and never keep binary blobs/images here — those go to object storage.
metadata:
  aep:
    kind: custom
---

# Database (our org standard: platform-managed PostgreSQL)

## What this skill does

When a service needs to persist its **own** structured data — the records, rows, and
relationships it owns and that no other internal service owns — our organization uses
a **platform-managed PostgreSQL** database, provisioned by the platform
(CloudNativePG) and declared as a **platform-resource dependency**. This skill stops
the agent from reaching for an embedded SQLite file, a local/in-memory store, or a
hand-rolled data file — all of which are ephemeral and vanish when the pod restarts.

## When to apply

Apply this skill when a service owns structured data that must outlive a request and
survive restarts:

- domain **entities / records / rows** the app creates (e.g. orders, tickets,
  comments, status-change history),
- **relationships** between them,
- anything the app must **query, list, or update** later.

Do NOT use it for:

- **files / images / blobs** → those go to object storage (see the `object-storage`
  skill); keep only their object key + metadata in the database.
- data another **internal service already owns** (people, teams, products) → resolve
  those by reference from the existing service (see the `internal-services` skill);
  never copy them into this database.

## How to consume it

1. **Declare a platform-resource dependency** on the service that owns the data, with
   `resourceType: postgres-cnpg`. It is a `platform-resource` (provisioned by the
   platform), NOT an `external` third-party dependency.

2. **The platform provisions a dedicated PostgreSQL and injects its connection** as
   environment variables, bound from the resource's outputs. Read them from the
   environment — never hardcode a connection string or credentials:

   | Resource output | Meaning | Bind to env (suggested) |
   |---|---|---|
   | `host` | database host | `DB_HOST` |
   | `port` | `5432` | `DB_PORT` |
   | `dbname` | database name (`appdb`) | `DB_NAME` |
   | `user` | app username (secret) | `DB_USER` |
   | `password` | app password (secret) | `DB_PASSWORD` |

   Assemble the DSN / connection string at runtime from those env vars.

3. **Use a real PostgreSQL client / ORM** for the component's language (e.g. `gorm` +
   the postgres driver for Go, `pg` / Prisma for Node, `psycopg` / SQLAlchemy for
   Python). Do **NOT** use `modernc.org/sqlite`, a `.db` file, or an in-memory
   database.

## Conventions

- One owning service per database; it holds only that service's own data.
- Store object-storage keys + metadata, never file bytes.
- Store references (ids) to data owned by other internal services, never copies.
- Run schema migrations on startup; treat `postgres-cnpg` as the only datastore.
