// Package pg holds the few Postgres helpers every module shares: id checks and mapping
// constraint violations to client errors. Queries themselves live in each module.
package pg

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sakib412/ekman"
)

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidID reports whether s looks like a uuid. Check path ids with it before querying, so a
// malformed id is a 404 instead of a Postgres cast error.
func ValidID(s string) bool { return uuidRE.MatchString(s) }

// NotFound is the 404 for a missing or malformed id.
func NotFound(what string) error { return ekman.NotFound(what + " not found") }

// Code returns the SQLSTATE of a Postgres error, or "".
func Code(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// Constraint returns the violated constraint name, or "".
func Constraint(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.ConstraintName
	}
	return ""
}

// Unique reports a unique violation.
func Unique(err error) bool { return Code(err) == "23505" }

// ForeignKey reports a foreign key violation.
func ForeignKey(err error) bool { return Code(err) == "23503" }

// Check reports a CHECK constraint violation.
func Check(err error) bool { return Code(err) == "23514" }

// Friendly maps constraint violations to 409/422 errors with a stable code and leaves every
// other error untouched. taken is the message for a unique violation (e.g. "slug already used").
func Friendly(err error, taken string) error {
	switch {
	case err == nil:
		return nil
	case Unique(err):
		return ekman.Conflict("already_exists", taken)
	case ForeignKey(err):
		return ekman.Conflict("in_use", "this record is referenced by other records (or references a missing one)")
	case Check(err):
		return ekman.Unprocessable("invalid_input", "rejected by constraint "+Constraint(err))
	}
	return err
}

// Array scans a Postgres array (e.g. text[]) into dst, a pointer to a slice:
// rows.Scan(pg.Array(&tags)). A NULL array scans as nil. (A pgtype.Map caches scan plans and is
// not safe for concurrent use, so each call gets its own.)
func Array(dst any) sql.Scanner { return pgtype.NewMap().SQLScanner(dst) }

// JSONList decodes a jsonb array column; anything unreadable becomes an empty list.
func JSONList[T any](raw []byte) []T {
	var out []T
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []T{}
	}
	return out
}
