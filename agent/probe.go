// /home/krylon/go/src/github.com/blicero/donkey/agent/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-17 18:59:29 krylon>

package agent

import (
	"encoding/json"
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
}

// LoadProbe gathers the system load average.
type LoadProbe struct {
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
