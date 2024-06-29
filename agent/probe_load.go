// /home/krylon/go/src/github.com/blicero/donkey/agent/probe_load.go
// -*- mode: go; coding: utf-8; -*-
// Created on 28. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-29 15:16:08 krylon>

package agent

import (
	"encoding/json"
	"log"
	"sync/atomic"
	"time"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/logdomain"
	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
)

/*
#include <stdlib.h>
*/
import "C"

// LoadProbe gathers the system load average.
type LoadProbe struct {
	active  atomic.Bool
	recordQ chan<- model.Record
	log     *log.Logger
}

// CreateLoadProbe creates a Probe that collects the system load average periodically.
func CreateLoadProbe(q chan<- model.Record) (*LoadProbe, error) {
	var err error
	p := &LoadProbe{
		recordQ: q,
	}

	if p.log, err = common.GetLogger(logdomain.Probe); err != nil {
		return nil, err
	}

	return p, nil
} // func CreateLoadProbe(q chan<- model.Record) (*LoadProbe, error)

// Collect gathers the system load averages and wraps them in a Record.
func (p *LoadProbe) Collect() (*model.Record, error) {
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
} // func (p *LoadProbe) Collect() (*model.Record, error)

// Running returns true if the Probe is active.
func (p *LoadProbe) Running() bool {
	return p.active.Load()
} // func (p *LoadProbe) Running() bool

// Stop tells the Probe to stop.
func (p *LoadProbe) Stop() {
	p.active.Store(false)
} // func (p *LoadProbe) Stop()

// Run executes the Probe's collect loop, this is usually executed in a separate goroutine.
func (p *LoadProbe) Run() {
	p.active.Store(true)
	defer p.active.Store(false)

	var ticker = time.NewTicker(ckInterval)
	defer ticker.Stop()

	for p.active.Load() {
		var (
			err error
			rec *model.Record
		)

		<-ticker.C
		if rec, err = p.Collect(); err != nil {
			p.log.Printf("[ERROR] Failed to collect Load Average: %s\n",
				err.Error())
		} else {
			p.recordQ <- *rec
		}
	}
} // func (p *LoadProbe) Run()
