// /home/krylon/go/src/github.com/blicero/donkey/database/01_database_host_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-23 17:50:35 krylon>

package database

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
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

} // func TestHostAdd(t *testing.T)

func TestRecordAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	const recCnt = 25
	var (
		err   error
		hosts []model.Host
	)

	if hosts, err = tdb.HostGetAll(); err != nil {
		t.Fatalf("Error fetching all hosts: %s", err.Error())
	}

	for _, h := range hosts {
		var stamp = time.Date(2024, 4, 1, 8, 15, 0, 0, time.Local)

		for i := 0; i < recCnt; i++ {
			var (
				load = [3]float64{
					rand.Float64() * 10,
					rand.Float64() * 10,
					rand.Float64() * 10,
				}
				rec = model.Record{
					HostID:    int64(h.ID),
					Timestamp: stamp.Add(time.Minute * time.Duration(i)),
					Source:    recordtype.LoadAvg,
					Payload: fmt.Sprintf(
						"[%.2f, %.2f, %.2f]",
						load[0],
						load[1],
						load[2]),
				}
			)

			if err = tdb.RecordAdd(&rec); err != nil {
				t.Errorf("Error adding record #%d for Host %s to database: %s",
					i,
					h.Name,
					err.Error())
			} else if rec.ID == 0 {
				t.Errorf("Record %d for host %s has no valid ID after adding",
					i,
					h.Name)
			}
		}
	}
} // func TestRecordAdd(t *testing.T)
