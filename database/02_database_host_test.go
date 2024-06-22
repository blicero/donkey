// /home/krylon/go/src/github.com/blicero/donkey/database/01_database_host_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-07 18:08:30 krylon>

package database

import (
	"testing"

	"github.com/blicero/donkey/model"
)

func TestHostAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	type testCase struct {
		h           model.Host
		expectError bool
	}

	var tests = []testCase{
		{
			h: model.Host{
				Name: "abobo",
				Addr: "192.168.0.1",
			},
		},
		{
			h: model.Host{
				Name: "",
				Addr: "192.168.0.2",
			},
			expectError: true,
		},
		{
			h: model.Host{
				Name: "cbobo",
				Addr: "",
			},
			expectError: true,
		},
	}

	for _, c := range tests {
		var err = tdb.HostAdd(&c.h)

		if err != nil {
			if !c.expectError {
				t.Errorf("Unexpected error trying to add host %s/%s: %s",
					c.h.Name,
					c.h.Addr,
					err.Error())
			}
		} else if c.expectError {
			t.Errorf("Trying to add host %s/%s should have resulted in an error but didn't",
				c.h.Name,
				c.h.Addr)
		} else if c.h.ID == 0 {
			t.Errorf("After adding, host %s/%s should have a valid ID",
				c.h.Name,
				c.h.Addr)
		}
	}

}
