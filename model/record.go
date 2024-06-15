// /home/krylon/go/src/github.com/blicero/donkey/model/record.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 15:46:51 krylon>

package model

import (
	"time"

	"github.com/blicero/donkey/model/recordtype"
)

// Record carries the data gathered by an Agent.
type Record struct {
	ID        int64
	Timestamp time.Time
	Source    recordtype.ID
	Data      map[string]string
}
