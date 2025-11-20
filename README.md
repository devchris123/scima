# scima

Schema migrations tool (Go) targeting multi-database support starting with SAP HANA.

## Features (initial)
- Versioned SQL migrations (up/down) stored as files: `NNNN_description.up.sql` and `NNNN_description.down.sql`
- Tracks applied versions in a table `schema_migrations`
- CLI commands: init, status, up, down
- Pluggable dialect interface (HANA first)
- Structured logging and observability hooks

## Quick start

```bash
# Set environment variables or use flags
scima init --driver hana --dsn "hdb://user:pass@host:30015" --migrations-dir ./migrations
scima up --driver hana --dsn "hdb://user:pass@host:30015" --migrations-dir ./migrations
scima status --driver hana --dsn "hdb://user:pass@host:30015" --migrations-dir ./migrations
```

## Migration files
Create paired files for each version:
```
0010_create_users_table.up.sql
0010_create_users_table.down.sql
```

## Future roadmap
### Near-term enhancements
1. HTTP API wrapper: expose endpoints `/status`, `/up`, `/down` allowing remote orchestration; use the same internal migrator package.
2. Multi-tenancy: strategy options
	- Separate schemas/databases per tenant (pass tenant DSN). Maintain a migration state table per tenant.
	- Single database with tenant-specific migration table names: `schema_migrations_<tenant>`.
	Provide an abstraction: `TenantProvider` enumerating active tenants; loop applying migrator logic.
3. Non-SQL migration formats: introduce interface `ExecutableMigration` allowing Go-based transformations or a declarative YAML -> generated SQL.
4. Additional dialects: PostgreSQL, MySQL, SQLite. Implement their `EnsureMigrationTable` and DML specifics (placeholder syntax differences).
5. Embedded migrations: use Go 1.22 `embed` package for packaging migrations into binary; precedence rules between disk and embedded.
6. Observability: add events channel + optional Prometheus counters (`scima_migrations_applied_total`, timings) and OpenTelemetry tracing around each statement.

### Longer-term ideas
- Automatic diff-based migration generation (introspect schema, produce delta SQL).
- Rollback safety analysis (flag irreversible statements like DROP COLUMN without data copy).
- Pluggable concurrency lock (advisory lock or lock table) to prevent double-run.
- Dry-run planner output (list statements without execution).
- Guardrails for production (confirmation prompts, window scheduling).

## Development
Run tests:
```bash
go test ./...
```

Run example applying sample migrations (requires valid DSN):
```bash
scima status --driver hana --dsn "$HANA_DSN"
scima up --driver hana --dsn "$HANA_DSN"
scima status --driver hana --dsn "$HANA_DSN"
```

Revert last migration:
```bash
scima down --driver hana --dsn "$HANA_DSN" --steps 1
```

## Contributing
PRs welcome. Add tests next to code files (`*_test.go`).
