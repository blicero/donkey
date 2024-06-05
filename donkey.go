// /home/krylon/go/src/github.com/blicero/donkey/donkey.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-05 19:21:44 krylon>

package main

import (
	"fmt"

	"github.com/blicero/donkey/common"
)

func main() {
	fmt.Printf("%s %s, built on %s",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))
}
