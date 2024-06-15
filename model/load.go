// /home/krylon/go/src/github.com/blicero/donkey/model/load.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 17:14:28 krylon>

package model

import (
	"encoding/json"
	"time"

	"github.com/blicero/krylib"
)

// Load is a record of the system load average that is available on most
// Unix-like systems (all that I have seen so far).
type Load struct {
	ID        krylib.ID
	Timestamp time.Time
	HostID    krylib.ID
	Load      [3]float64
}

// Payload returns the Load record's payload as a JSON string.
func (l *Load) Payload() string {
	var (
		err error
		buf []byte
	)

	if buf, err = json.Marshal(l.Load); err != nil {
		panic(err)
	}

	return string(buf)
}
