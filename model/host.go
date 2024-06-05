// /home/krylon/go/src/github.com/blicero/donkey/model/host.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-05 18:05:06 krylon>

package model

import "github.com/blicero/krylib"

type Host struct {
	ID      krylib.ID
	Name    string
	Address string
}
