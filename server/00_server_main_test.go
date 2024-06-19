// /home/krylon/go/src/github.com/blicero/donkey/server/00_server_test_main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-19 19:03:06 krylon>

package server

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/database"
	"github.com/blicero/donkey/model"
)

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/donkey_server_test_20060102_150405")
	)

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
			err.Error())
		os.Exit(1)
	} else if err = prepare_db(); err != nil {
		fmt.Fprintf(os.Stderr,
			"Failed to initialize database: %s\n",
			err.Error())
		os.Exit(1)
	} else if result = m.Run(); result == 0 {
		// If any test failed, we keep the test directory (and the
		// database inside it) around, so we can manually inspect it
		// if needed.
		// If all tests pass, OTOH, we can safely remove the directory.
		fmt.Printf("Removing BaseDir %s\n",
			baseDir)
		_ = os.RemoveAll(baseDir)
	} else {
		fmt.Printf(">>> TEST DIRECTORY: %s\n", baseDir)
	}

	os.Exit(result)
} // func TestMain(m *testing.M)

func prepare_db() error {
	var (
		err error
		db  *database.Database
	)

	var test_hosts = []model.Host{
		{
			Name: "abobo",
			Addr: "10.10.99.1",
			OS:   "Debian",
		},
		{
			Name: "bbobo",
			Addr: "10.10.99.2",
			OS:   "FreeBSD",
		},
		{
			Name: "cbobo",
			Addr: "10.10.99.3",
			OS:   "OpenBSD",
		},
	}

	if db, err = database.Open(common.DbPath); err != nil {
		return err
	} else if err = db.Begin(); err != nil {
		return err
	}

	for _, h := range test_hosts {
		if err = db.HostAdd(&h); err != nil {
			db.Rollback()
			return err
		}
	}

	if err = db.Commit(); err != nil {
		return err
	}

	db.Close()

	return nil
} // func prepare_db() error
