// /home/krylon/go/src/github.com/blicero/donkey/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-07 18:38:22 krylon>

package logdomain

//go:generate stringer -type=ID

// ID is an id...
type ID uint8

// These constants represent the pieces of the application that need to log stuff.
const (
	Common ID = iota
	Client
	DBPool
	Database
	Server
)

// AllDomains returns a slice of all the valid values for ID.
func AllDomains() []ID {
	return []ID{
		Common,
		Client,
		DBPool,
		Database,
		Server,
	}
} // func AllDomains() []ID
