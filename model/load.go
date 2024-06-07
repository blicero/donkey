// /home/krylon/go/src/github.com/blicero/donkey/model/load.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-06 20:30:12 krylon>

package model

import (
	"time"

	"github.com/blicero/krylib"
)

type Load struct {
	ID        krylib.ID
	Timestamp time.Time
	HostID    krylib.ID
	Load      [3]float64
}
