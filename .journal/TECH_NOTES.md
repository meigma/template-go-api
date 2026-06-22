# Technical Notes

- Use hexagonal architecture at all times. Keep business logic isolated from CLI, filesystem, network, storage, and other external adapters.
- Prefer functional testing before calling any feature complete. Unit tests are useful, but they do not prove the tool works the way the design intends.
- Take an agile approach to development. Avoid waterfall: underspecify when useful, prototype early, learn from the result, and refine from working behavior.
- Target shape for the API-server transformation lives in `.journal/TARGET_SHAPE.md` (temporary design guide). Decided in session 001: chi v5 on net/http; Huma (code-first OpenAPI, transport-scoped) + MkDocs neoteroi OAD; pragmatic ports & adapters; slog + OpenMetrics `/metrics`; full in-memory reference slice with `/healthz` + `/readyz`; Cobra root (`serve`/`version`/`openapi`).
