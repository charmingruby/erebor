# Erebor

Dwarf-crafted building blocks for shared infrastructure.

## What is this?

Reusable infrastructure building blocks for Go projects. Import what you need, keep business logic in your application, and avoid rebuilding the same foundation across services.

## Getting Started

Requires Go 1.27 or later.

Add `github.com/charmingruby/erebor` as a dependency, then import the packages your application needs.

## Packages

- [Discord](discord) — Slash commands and mention-driven bots with bounded concurrency, safe replies, and graceful shutdown.
- [Environment](env) — Typed configuration loading from environment variables, with optional `.env` file support.
- [HTTP](httpx) — HTTP clients, servers, JSON request validation, response helpers, logging, recovery, and health probes.
- [AI](aikit) — Model inference, agents, validated tools, embeddings, and MCP servers over local stdio or remote Streamable HTTP, built on Genkit.
- [AI providers](aikit/provider) — Google AI and OpenAI integrations.
- [Logging](o11y/logging) — Structured JSON logging and context-aware logger access.
- [PostgreSQL](store/postgrex) — Connections, named prepared statements, typed query helpers, and transactions.
- [Validation](validator) — Struct validation with translated, field-level errors.
