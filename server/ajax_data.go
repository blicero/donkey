// /home/krylon/go/src/github.com/blicero/donkey/server/ajax_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-10 22:24:41 krylon>

package server

import "time"

type response struct {
	Status    bool
	Message   string
	Timestamp time.Time
}
