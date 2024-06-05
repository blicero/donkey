// /home/krylon/go/src/github.com/blicero/donkey/model/load.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-05 18:23:28 krylon>

package model

import "fmt"

type Load [3]float64

func (l Load) String() string {
	return fmt.Sprintf("{%.1f / %.1f / %.1f}",
		l[0],
		l[1],
		l[2],
	)
} // func (l Load) String() string
