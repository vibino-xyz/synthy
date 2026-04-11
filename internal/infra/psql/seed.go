package psql

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stable IDs for seed data — exactly 32 characters each.
const (
	SeedUserID = "seed_user_0000000000000000000001"
	SeedOrgID  = "seed_org_00000000000000000000001"
)

// EnsureSeedData inserts a seed user and organization if they don't already
// exist (idempotent via ON CONFLICT DO NOTHING). Returns the seed org ID.
func EnsureSeedData(ctx context.Context, db *pgxpool.Pool) (string, error) {
	now := time.Now().UTC()

	const insertUser = `
		INSERT INTO "user" (id, name, email, username, password_hash, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO NOTHING`

	if _, err := db.Exec(ctx, insertUser,
		SeedUserID, "Seed User", "seed@synthy.local", "seed_user",
		"seed-password-hash-not-real", true, now, now,
	); err != nil {
		return "", err
	}

	const insertOrg = `
		INSERT INTO organization (id, name, slug, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO NOTHING`

	if _, err := db.Exec(ctx, insertOrg,
		SeedOrgID, "Seed Org", "seed-org", SeedUserID, now, now,
	); err != nil {
		return "", err
	}

	return SeedOrgID, nil
}
