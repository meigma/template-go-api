---
id: 007
title: Explore restructuring internal/ to couple domain code
started: 2026-06-23
---

## 2026-06-23 15:12 — Kickoff
Goal for the session: explore how plausible it is to refactor the current
`internal/` package structure toward coupling domain code into one logical
package hierarchy, rather than the present split across multiple top-level
packages. This is an exploration/feasibility session — no implementation yet.

Current state of the world: the template is fully built and merged through PR #6
(`18b56e7`). `internal/` currently splits a single domain (`todo`) across several
top-level packages following pragmatic ports & adapters:
- domain: `internal/todo`
- adapters: `internal/adapter/{memory,http,postgres}` (+ `http/middleware`,
  `http/problem`, `http/todoapi`)
- cross-cutting: `internal/{config,observability,logctx,app,cli,integration}`

The user's framing: today a domain's code (e.g. `todo`) is spread across
`internal/todo`, `internal/adapter/memory`, `internal/adapter/http/todoapi`,
`internal/adapter/postgres`, etc. The question is whether to instead group all of
a domain's code under one logical hierarchy (e.g. everything `todo`-related
nested together) and how plausible/desirable that is for the template.

Plan: paused after session setup per the user's instruction. Awaiting the user's
detailed framing and constraints before exploring options.
