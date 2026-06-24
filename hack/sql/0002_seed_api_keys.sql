-- ############################################################################
-- # INSECURE MOCK API KEYS — LOCAL DEVELOPMENT ONLY. DO NOT USE IN PRODUCTION.
-- ############################################################################
--
-- These are plaintext placeholder credentials seeded into the EPHEMERAL Docker
-- Compose database so `docker compose up` demonstrates the authorization tier
-- end to end with zero config. They are NOT a security mechanism:
--
--   * The values are public, hard-coded, and committed to the repository.
--   * They are stored verbatim (the shipped store matches keys as plaintext).
--   * They are data, not schema: real deployments run the goose migrations but
--     NEVER apply hack/sql/, so these mock keys can never reach a real database.
--
-- Remove this file (and replace the shipped API-key authenticator with real
-- authn) before going to production — see DELETE_ME.md. Real deployments insert
-- their own api_keys rows out of band.
--
-- The roles below must match the shipped policies: Role::"user" satisfies the
-- todo slice's coarse policy (internal/todo/authz/policy.cedar) and Role::"admin"
-- satisfies the cross-cutting admin override (internal/authz/base.cedar).
--
-- Columns are (key, subject, roles text[]) per migration
-- internal/adapter/postgres/migrations/00002_create_api_keys.sql. The
-- authenticator reads the key from the X-API-Key header or an
-- Authorization: Bearer credential.
INSERT INTO api_keys (key, subject, roles) VALUES
  ('dev-user-key',  'dev-user',  ARRAY['user']),
  ('dev-admin-key', 'dev-admin', ARRAY['admin'])
ON CONFLICT (key) DO NOTHING;
