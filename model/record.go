// /home/krylon/go/src/github.com/blicero/donkey/model/record.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-14 21:40:28 krylon>

package model

import (
	"time"
)

// Record carries the data gathered by an Agent.
type Record struct {
	ID        int64
	Timestamp time.Time
	Source    string
	Data      map[string]string
}
