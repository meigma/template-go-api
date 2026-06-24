package apikey

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// lookupQuery resolves an API key to its subject and roles. It is hand-written
// and parameterized (so the key is never interpolated) to keep this package
// self-contained and trivially removable — it deliberately does not introduce a
// second sqlc package.
//
// SECURITY: this day-one implementation stores and matches keys verbatim. The
// production hardening path is to store only a hash of the key (for example
// SHA-256) and compare in constant time (crypto/subtle.ConstantTimeCompare)
// after hashing the presented key, so a leaked table dump does not reveal usable
// credentials and lookups are not timing-distinguishable. See DELETE_ME.
const lookupQuery = `SELECT subject, roles FROM api_keys WHERE key = $1`

// Store is the PostgreSQL-backed APIKeyStore. It resolves keys against the
// api_keys table using the shared pgx pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a Store over the shared connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Lookup resolves key to its subject and roles. It returns (Identity, false, nil)
// when no row matches (an unknown key, not an error) and a non-nil error only on
// a query failure. The key is passed as a bind parameter and never logged.
func (s *Store) Lookup(ctx context.Context, key string) (Identity, bool, error) {
	var (
		subject string
		roles   []string
	)

	err := s.pool.QueryRow(ctx, lookupQuery, key).Scan(&subject, &roles)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, false, nil
		}

		// Do not include key in the error: it must never reach a log line.
		return Identity{}, false, fmt.Errorf("query api key: %w", err)
	}

	return Identity{Subject: subject, Roles: roles}, true, nil
}
