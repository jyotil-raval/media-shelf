package db

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS media_items (
	id          SERIAL       PRIMARY KEY,
	title       TEXT         NOT NULL,
	media_type  TEXT         NOT NULL,
	sub_type    TEXT,
	source      TEXT         NOT NULL,
	source_id   TEXT,
	status      TEXT         NOT NULL,
	score       INTEGER,
	progress    INTEGER,
	total       INTEGER,
	notes       TEXT,
	created_at  TIMESTAMPTZ  DEFAULT NOW(),
	updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_source ON media_items(source, source_id);
`

func Migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
