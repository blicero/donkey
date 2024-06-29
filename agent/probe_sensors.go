// /home/krylon/go/src/github.com/blicero/donkey/agent/probe_sensors.go
// -*- mode: go; coding: utf-8; -*-
// Created on 28. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-29 15:17:19 krylon>

package agent

import (
	"bytes"
	"log"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/logdomain"
	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
)

const prog = "sensors"

// SensorsProbe gathers data from sensors, most importantly temperature.
type SensorsProbe struct {
	active  atomic.Bool
	recordQ chan<- model.Record
	log     *log.Logger
}

// CreateSensorsProbe creates a Probe that queries the sensors attached to the system periodically.
func CreateSensorsProbe(recQ chan<- model.Record) (*SensorsProbe, error) {
	var err error
	p := &SensorsProbe{
		recordQ: recQ,
	}

	if p.log, err = common.GetLogger(logdomain.Probe); err != nil {
		return nil, err
	}

	return p, nil
} // func CreateSensorsProbe() (*SensorsProbe, error)

// Collect data from the sensors
func (p *SensorsProbe) Collect() (*model.Record, error) {
	var (
		err            error
		bufOut, bufErr bytes.Buffer
		cmd            *exec.Cmd
		rec            = &model.Record{Source: recordtype.Sensors}
	)

	cmd = exec.Command(prog, "-j")
	cmd.Stdout = &bufOut
	cmd.Stderr = &bufErr

	if err = cmd.Run(); err != nil {
		return nil, err
	}

	rec.Timestamp = time.Now()
	rec.Payload = bufOut.String()

	return rec, nil
} // func (p *SensorsProbe) Collect() (*model.Record, error)

// Running returns the Probe's active flag
func (p *SensorsProbe) Running() bool {
	return p.active.Load()
}

// Stop clears the Probe's active flag
func (p *SensorsProbe) Stop() {
	p.active.Store(false)
}

// Run executes the Probe's collect loop, this is usually executed in a separate goroutine.
func (p *SensorsProbe) Run() {
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
			p.log.Printf("[ERROR] Failed to collect record: %s\n",
				err.Error())
		} else {
			p.recordQ <- *rec
		}
	}
} // func (p *LoadProbe) Run()
