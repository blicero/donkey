// /home/krylon/go/src/github.com/blicero/donkey/server/01_server_test_server.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-25 14:47:06 krylon>

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/blicero/donkey/model"
	"github.com/blicero/donkey/model/recordtype"
)

const testAddr = "[::1]:4197"

var srv *Server

func TestCreateServer(t *testing.T) {
	var err error

	if srv, err = Create(testAddr); err != nil {
		srv = nil
		t.Fatalf("Failed to create Server: %s", err.Error())
	}

	go srv.Run()

	time.Sleep(time.Millisecond * 250)
} // func TestCreateServer(t *testing.T)

func TestReportData(t *testing.T) {
	const path = "/ws/report"

	if srv == nil {
		t.SkipNow()
	}

	var (
		err    error
		client http.Client
		addr   = fmt.Sprintf("http://%s%s",
			testAddr,
			path)
	)

	for _, h := range testHosts {
		var (
			req        *http.Request
			res        *http.Response
			reply      model.Response
			rec        model.Record
			serialized []byte
			buf        *bytes.Buffer
		)

		rec = model.Record{
			HostID:    int64(h.ID),
			Timestamp: time.Now(),
			Source:    recordtype.LoadAvg,
			Payload:   "[1.15, 3.17, 5.2]",
		}

		if serialized, err = json.Marshal(&rec); err != nil {
			t.Errorf("Failed to serialized Record: %s", err.Error())
			continue
		}

		buf = bytes.NewBuffer(serialized)

		if req, err = http.NewRequest("POST", addr, buf); err != nil {
			t.Errorf("Failed to create HTTP request for host %d: %s",
				h.ID,
				err.Error())
			continue
		} else if res, err = client.Do(req); err != nil {
			t.Errorf("Failed to perform HTTP request for %s: %s",
				addr,
				err.Error())
			continue
		}

		defer res.Body.Close()
		buf.Reset()

		if _, err = io.Copy(buf, res.Body); err != nil {
			t.Errorf("Failed to read response body from Server: %s",
				err.Error())
			continue
		} else if err = json.Unmarshal(buf.Bytes(), &reply); err != nil {
			t.Errorf("Failed to unmarshal response body: %s\n\n%s\n",
				err.Error(),
				buf.String())
			continue
		}
	}
} // func TestReportData(t *testing.T)
