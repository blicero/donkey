// /home/krylon/go/src/github.com/blicero/donkey/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 16:13:15 krylon>

package database

// With my new approach of storing JSON in the database, I probably don't need
// the load table, do I?

var qInit = []string{
	`
CREATE TABLE host (
    id		INTEGER PRIMARY KEY,
    name	TEXT NOT NULL,
    addr	TEXT NOT NULL,
    os          TEXT NOT NULL DEFAULT '',
    last_contact INTEGER NOT NULL DEFAULT 0,
    UNIQUE (name, addr),
    CHECK (name <> '' AND addr <> '')
) STRICT
`,
	"CREATE INDEX host_addr_idx ON host (addr)",
	"CREATE INDEX host_name_idx ON host (name)",
	"CREATE INDEX host_contact_idx ON host (last_contact)",

	`
CREATE TABLE load (
    id INTEGER PRIMARY KEY,
    host_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    load1 REAL NOT NULL,
    load5 REAL NOT NULL,
    load15 REAL NOT NULL,
    FOREIGN KEY (host_id) REFERENCES host (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE,
    UNIQUE (host_id, timestamp),
    CHECK (load1 >= 0.0 AND load5 >= 0.0 AND load15 >= 0.0)
) STRICT
`,
	"CREATE INDEX load_host_idx ON load (host_id)",
	"CREATE INDEX load_stamp_idx ON load (timestamp)",

	`
CREATE TABLE record (
    id INTEGER PRIMARY KEY,
    host_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    recordtype INTEGER NOT NULL,
    payload TEXT NOT NULL,
    UNIQUE (host_id, timestamp, recordtype),
    FOREIGN KEY (host_id) REFERENCES host (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX record_host_idx ON record (host_id)",
	"CREATE INDEX record_time_idx ON record (timestamp)",
	"CREATE INDEX record_type_idx ON record (recordtype)",
}
