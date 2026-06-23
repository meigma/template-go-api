# Technical Notes

- Use hexagonal architecture at all times. Keep business logic isolated from CLI, filesystem, network, storage, and other external adapters.
- Prefer functional testing before calling any feature complete. Unit tests are useful, but they do not prove the tool works the way the design intends.
- Take an agile approach to development. Avoid waterfall: underspecify when useful, prototype early, learn from the result, and refine from working behavior.
- The API-server template is **built** (slices 1–2 merged: PR #4 `745a9ed`, PR #5 `05f5446`). The code + `README.md` are the source of truth for architecture and usage; `.journal/TARGET_SHAPE.md` is the original design doc, now largely realized (historical). Shape as built: chi v5 + Huma v2 (transport-scoped, code-first OpenAPI 3.0.3); pragmatic ports & adapters; domain `internal/todo`, adapters under `internal/adapter/{memory,http}` (+ `http/middleware`, `http/problem`, `http/todoapi`), `internal/{config,observability,logctx,app,cli}`; slog + Prometheus `/metrics` on a **dedicated listener** (`--metrics-addr`, default `:9090`); RFC 9457 on every non-Huma surface; OpenAPI exported server-less → neoteroi OAD render with a `root:check` drift-guard.
- Convention: after changing the API, run `moon run openapi` to refresh the committed `docs/docs/openapi.yaml`; CI (`root:check` → `openapi-check`) fails if it drifts.
- Future-slice seams left open (not yet built): authn/authz; Postgres adapter + testcontainers; OTel tracing; rate limiting; pagination; API versioning; mockery.
