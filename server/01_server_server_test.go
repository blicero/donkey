// /home/krylon/go/src/github.com/blicero/donkey/server/01_server_test_server.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-20 13:29:14 krylon>

package server

import "testing"

const testAddr = "[::1]:4197"

var srv *Server

func TestCreateServer(t *testing.T) {
	var err error

	if srv, err = Create(testAddr); err != nil {
		srv = nil
		t.Fatalf("Failed to create Server: %s", err.Error())
	}
} // func TestCreateServer(t *testing.T)
