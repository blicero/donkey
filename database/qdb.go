// /home/krylon/go/src/github.com/blicero/donkey/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-05 18:53:17 krylon>

package database

import "github.com/blicero/donkey/database/query"

var qDB = map[query.ID]string{
	query.HostAdd:           "INSERT INTO host (name, addr) VALUES (?, ?) RETURNING id",
	query.HostGetByID:       "SELECT name, addr FROM host WHERE id = ?",
	query.HostGetByAddr:     "SELECT id, name FROM host WHERE addr = ?",
	query.HostGetByName:     "SELECT id, addr FROM host WHERE name = ?",
	query.HostGetAll:        "SELECT id, name, addr FROM host",
	query.HostDelete:        "DELETE FROM host WHERE id = ?",
	query.HostUpdateName:    "UPDATE host SET name = ? WHERE id = ?",
	query.HostUpdateAddress: "UPDATE host SET addr = ? WHERE id = ?",
	query.LoadAdd: `
INSERT INTO load (host_id, timestamp, load1, load5, load15)
          VALUES (      ?,         ?,     ?,     ?,      ?)
RETURNING id
`,
	query.LoadGetByHost: "SELECT id, timestamp, load1, load5, load15 FROM load WHERE host_id = ? ORDER BY timestamp",
	query.LoadgetByPeriod: `
SELECT
    id,
    host_id,
    timestamp,
    load1,
    load5,
    load15
FROM load
WHERE timestamp BETWEEN ? AND ?
ORDER BY timestamp, host_id
`,
}
