# ADR 0001: Modular Monolith

Status: accepted

The backend is one deployable application with feature-owned packages. Modules expose application methods and receive dependencies through constructors. Domain types do not import HTTP, pgx or Redis packages. API and worker entry points share application code but have separate process lifecycles.

This avoids distributed transactions and preserves explicit module boundaries while allowing independent tests and later extraction.
