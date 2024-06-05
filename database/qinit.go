// /home/krylon/go/src/github.com/blicero/donkey/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-05 18:52:21 krylon>

package database

var qInit []string = []string{
	`
CREATE TABLE host (
    id		INTEGER PRIMARY KEY,
    name	TEXT NOT NULL,
    addr	TEXT NOT NULL,
    UNIQUE (name, address)
) STRICT
`,
	"CREATE INDEX host_addr_idx ON host (addr)",
	"CREATE INDEX host_name_idx ON host (name)",

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
}
