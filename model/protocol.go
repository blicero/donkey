// /home/krylon/go/src/github.com/blicero/donkey/model/protocol.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-12 19:29:09 krylon>

package model

import "time"

// Response is what the Server sends to the Agent after handling a request.
type Response struct {
	Status    bool
	Message   string
	Timestamp time.Time
}
