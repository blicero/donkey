// /home/krylon/go/src/github.com/blicero/donkey/agent/agent.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-11 23:19:47 krylon>

// Package agent implements the client side of the application.
package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/logdomain"
)

type config struct {
	Server string
	HostID int64
}

// Agent wraps the state of the client.
type Agent struct {
	server string
	hostID int64
	name   string
	log    *log.Logger
	client http.Client // nolint: unused,deadcode
	os     string
}

// Create creates a new Agent.
func Create(srv string) (*Agent, error) {
	var (
		err     error
		ag      = &Agent{server: srv}
		version string
	)

	if ag.log, err = common.GetLogger(logdomain.Agent); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Logger for %s: %s\n",
			logdomain.Agent,
			err.Error())
		return nil, err
	} else if ag.os, version, err = DetectOS(); err != nil {
		ag.log.Printf("[ERROR] Failed to detect operating system: %s\n",
			err.Error())
		return nil, err
	} else if ag.name, err = os.Hostname(); err != nil {
		ag.log.Printf("[ERROR] Failed to ask OS for hostname: %s\n",
			err.Error())
		return nil, err
	} else if err = ag.readConfig(); err != nil {
		ag.log.Printf("[ERROR] Could not process configuration file: %s\n",
			err.Error())
		return nil, err
	}

	ag.log.Printf("[DEBUG] Agent coming up on %s, running %s %s\n",
		ag.name,
		ag.os,
		version)

	return ag, nil
} // func Create(srv string) (*Agent, error)

func (ag *Agent) readConfig() error {
	var (
		fh  *os.File
		cfg config
		buf bytes.Buffer
		err error
	)

	if fh, err = os.Open(common.AgentConfPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {

		}
		ag.log.Printf("[ERROR] Cannot open agent config %s: %s\n",
			common.AgentConfPath,
			err.Error())
		return err
	}

	defer fh.Close()

	if _, err = io.Copy(&buf, fh); err != nil {
		ag.log.Printf("[ERROR] Failed to read from %s: %s\n",
			common.AgentConfPath,
			err.Error())
		return err
	} else if err = json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		ag.log.Printf("[ERROR] Error decoding config: %s\n",
			err.Error())
		return err
	} else if cfg.HostID != 0 {
		ag.hostID = cfg.HostID
	}

	ag.server = cfg.Server
	return nil
} // func (ag *Agent) readConfig() error
