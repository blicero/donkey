// /home/krylon/go/src/github.com/blicero/donkey/model/payload.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 17:23:49 krylon>

package model

// Payload can convert their "juicy bits" to JSON and return them as a string.
type Payload interface {
	Payload() string
}
