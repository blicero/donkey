// /home/krylon/go/src/github.com/blicero/donkey/model/recordtype/recordtype.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 15:46:21 krylon>

// Package recordtype provides symbolic constants to identify the different
// types of Record Agents collect and forward to the Server.
package recordtype

//go:generate stringer -type=ID

type ID uint8

const (
	LoadAvg ID = iota
	Sensors
	CPUFreq
	RAM
)
