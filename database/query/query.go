// /home/krylon/go/src/github.com/blicero/donkey/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-07 18:22:15 krylon>

//go:generate stringer -type=ID

// Package query defines symbolic constants to reference database queries.
package query

// ID Represents a database query.
type ID uint8

const (
	HostAdd ID = iota
	HostGetByID
	HostGetByAddr
	HostGetByName
	HostGetAll
	HostDelete
	HostUpdateName
	HostUpdateAddr
	HostUpdateOS
	LoadAdd
	LoadGetByHost
	LoadgetByPeriod
)
