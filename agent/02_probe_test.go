// /home/krylon/go/src/github.com/blicero/donkey/agent/02_probe_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 17. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-07-01 18:51:01 krylon>

package agent

import (
	"testing"

	"github.com/blicero/donkey/model"
)

func TestLoadProbe(t *testing.T) {
	var (
		lp  Probe
		err error
		q   chan model.Record
		rec *model.Record
	)

	q = make(chan model.Record, 2)

	if lp, err = CreateLoadProbe(q); err != nil {
		t.Fatalf("Failed to create LoadProbe: %s",
			err.Error())
	} else if rec, err = lp.Collect(); err != nil {
		t.Errorf("Failed to collect load average: %s\n", err.Error())
	} else if rec == nil {
		t.Error("Collect() did not return an error, but the record was nil")
	}
} // func TestLoadProbe(t *testing.T)

func TestSensorsProbe(t *testing.T) {
	var (
		q   chan model.Record
		sp  Probe
		err error
		rec *model.Record
	)

	q = make(chan model.Record, 2)

	if sp, err = CreateSensorsProbe(q); err != nil {
		t.Fatalf("Failed to create SensorsProbe: %s",
			err.Error())
	} else if rec, err = sp.Collect(); err != nil {
		t.Errorf("Failed to collect load average: %s\n", err.Error())
	} else if rec == nil {
		t.Error("Collect() did not return an error, but the record was nil")
	}
} // func TestSensorsProbe(t *testing.T)
