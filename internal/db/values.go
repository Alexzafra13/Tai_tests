package db

import (
	"errors"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// TimeFormat is how every table stores timestamps: UTC with milliseconds,
// so they compare and sort correctly as text.
const TimeFormat = "2006-01-02T15:04:05.000Z"

func Timestamp(t time.Time) string { return t.UTC().Format(TimeFormat) }

// IsUnique reports whether err is a UNIQUE constraint failure.
func IsUnique(err error) bool { return hasCode(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE) }

// IsForeignKey reports whether err is a FOREIGN KEY constraint failure:
// a referenced row is missing, or a row still referenced was deleted.
func IsForeignKey(err error) bool { return hasCode(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY) }

func hasCode(err error, code int) bool {
	var e *sqlite.Error
	return errors.As(err, &e) && e.Code() == code
}
