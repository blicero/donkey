// /home/krylon/go/src/github.com/blicero/donkey/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-17 22:41:49 krylon>

package database

import "github.com/blicero/donkey/database/query"

var qDB = map[query.ID]string{
	query.HostAdd:               "INSERT INTO host (name, addr, os) VALUES (?, ?, ?) RETURNING id",
	query.HostGetByID:           "SELECT name, addr, os, last_contact FROM host WHERE id = ?",
	query.HostGetByAddr:         "SELECT id, name, os, last_contact FROM host WHERE addr = ?",
	query.HostGetByName:         "SELECT id, addr, os, last_contact FROM host WHERE name = ?",
	query.HostGetAll:            "SELECT id, name, addr, os, last_contact FROM host",
	query.HostDelete:            "DELETE FROM host WHERE id = ?",
	query.HostUpdateName:        "UPDATE host SET name = ? WHERE id = ?",
	query.HostUpdateAddr:        "UPDATE host SET addr = ? WHERE id = ?",
	query.HostUpdateOS:          "UPDATE host SET os = ? WHERE id = ?",
	query.HostUpdateLastContact: "UPDATE host SET last_contact = ? WHERE id = ?",
	query.LoadAdd:               "INSERT INTO record (host_id, timestamp, recordtype, payload) VALUES (?, ?, ?, ?)",
	query.LoadGetByHost: `
SELECT
    id,
    timestamp,
    payload ->> '$[0]' AS load1,
    payload ->> '$[1]' AS load5,
    payload ->> '$[2]' AS load15
FROM record
WHERE host_id = ? AND recordtype = ?
ORDER BY timestamp
`,
	query.LoadgetByPeriod: `
SELECT
    id,
    host_id,
    timestamp,
    payload ->> '$[0]' AS load1,
    payload ->> '$[1]' AS load5,
    payload ->> '$[2]' AS load15
FROM record
WHERE recordtype = ? AND timestamp BETWEEN ? AND ?
ORDER BY timestamp, host_id
`,
	query.RecordAdd: "INSERT INTO record (host_id, timestamp, recordtype, payload) VALUES (?, ?, ?, ?)",
	query.RecordGetByHost: `
SELECT
    id,
    timestamp,
    recordtype,
    payload
FROM record
WHERE host_id = ?
ORDER BY timestamp, recordtype
`,
	query.RecordGetByType: `
SELECT
    id,
    host_id,
    timestamp,
    payload
FROM record
WHERE recordtype = ?
ORDER BY host_id, timestamp
`,
	query.RecordGetByHostType: `
SELECT
    id,
    timestamp,
    payload
FROM record
WHERE host_id = ? AND recordtype = ?
ORDER BY timestamp
`,
}
