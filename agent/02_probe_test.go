// /home/krylon/go/src/github.com/blicero/donkey/agent/02_probe_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 17. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-17 19:01:51 krylon>

package agent

import (
	"testing"

	"github.com/blicero/donkey/model"
)

func TestLoadProbe(t *testing.T) {
	var (
		lp  LoadProbe
		err error
		rec *model.Record
	)

	if rec, err = lp.Collect(); err != nil {
		t.Errorf("Failed to collect load average: %s\n", err.Error())
	} else if rec == nil {
		t.Error("Collect() did not return an error, but the record was nil")
	}
} // func TestLoadProbe(t *testing.T)
