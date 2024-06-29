// /home/krylon/go/src/github.com/blicero/donkey/agent/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-29 14:40:54 krylon>

package agent

import (
	"time"

	"github.com/blicero/donkey/model"
)

const ckInterval = time.Millisecond * 5000

// Probe defines an interface for components that gather data from the node and
// return them as Records that can be sent back to the Server.
type Probe interface {
	Collect() (*model.Record, error)
	Run()
	Running() bool
	Stop()
}
