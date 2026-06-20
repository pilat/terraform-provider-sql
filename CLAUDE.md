# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Terraform provider (`github.com/pilat/terraform-provider-sql`) exposing a single `sql` resource that runs `up` SQL on create and `down` SQL on destroy. PostgreSQL only — the driver is hardcoded to `postgres` (`lib/pq`).

## Commands

- Test: `go test ./...` (the only CI command; there is no Makefile)
- Release: handled by goreleaser on a `v*` git tag — do not run manually

## Stack

- Built on `terraform-plugin-sdk/v2` (v2.28.0), **not** the newer terraform-plugin-framework. Write schema/CRUD code in the SDK v2 style (`*schema.Resource`, `*schema.ResourceData`, `diag.Diagnostics`) — framework patterns do not apply here.

## Intentional design (do not "fix" these)

These look like bugs but are deliberate (see commits "Do not allow to change up attribute", "Ignore update handler"):

- `up` is immutable after create — `CustomizeDiff` rejects changes to it.
- `Update` performs no SQL; it only warns when `database`/`down` change. State updates, DB is untouched.
- `Read` is a no-op — the provider does not reconcile against actual DB state.
- Resource ID is the first 8 chars of `sha256(up)`.
- Multiline SQL is split on `;\n` / `;\r\n`, so statements must end with a semicolon followed by a newline.

## Notes

- `tests/` holds manual integration tests (docker-compose Postgres + a `.tf` config). Work in progress — not wired into CI; treat as scratch.
