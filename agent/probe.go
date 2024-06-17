// /home/krylon/go/src/github.com/blicero/donkey/agent/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-17 19:35:30 krylon>

package agent

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
)

/*
#include <stdlib.h>
*/
import "C"

// Probe defines an interface for components that gather data from the node and
// return them as Records that can be sent back to the Server.
type Probe interface {
	Collect() (*model.Record, error)
	Run()
	Running() bool
	Stop()
}

// LoadProbe gathers the system load average.
type LoadProbe struct {
	active  atomic.Bool
	recordQ chan model.Record
}

// Collect gathers the system load averages and wraps them in a Record.
func (lp *LoadProbe) Collect() (*model.Record, error) {
	var (
		err    error
		raw    [3]C.double
		data   [3]float64
		buf    []byte
		record *model.Record
	)

	C.getloadavg(&raw[0], 3)

	data[0] = float64(raw[0])
	data[1] = float64(raw[1])
	data[2] = float64(raw[2])

	if buf, err = json.Marshal(data); err != nil {
		return nil, err
	}

	record = &model.Record{
		Timestamp: time.Now(),
		Source:    recordtype.LoadAvg,
		Payload:   string(buf),
	}

	return record, nil
} // func (lp *LoadProbe) Collect() (*model.Record, error)

// Running returns true if the Probe is active.
func (lp *LoadProbe) Running() bool {
	return lp.active.Load()
} // func (lp *LoadProbe) Running() bool

// Stop tells the Probe to stop.
func (lp *LoadProbe) Stop() {
	lp.active.Store(false)
} // func (lp *LoadProbe) Stop()

// Run executes the Probe's collect loop, this is usually executed in a separate goroutine.
func (lp *LoadProbe) Run() {
	lp.active.Store(true)
	defer lp.active.Store(false)

	for lp.active.Load() {
		// do stuff...
	}
} // func (lp *LoadProbe) Run()
