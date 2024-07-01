// /home/krylon/go/src/github.com/blicero/donkey/agent/agent.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-07-01 19:37:44 krylon>

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
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/blicero/donkey/common"
	"github.com/blicero/donkey/logdomain"
	"github.com/blicero/donkey/model"
	"github.com/blicero/krylib"
)

const (
	heartbeat = time.Millisecond * 2500
)

type config struct {
	Server string
	HostID int64
	Probes map[string]int
}

// Agent wraps the state of the client.
type Agent struct {
	server  string
	hostID  krylib.ID
	name    string
	active  atomic.Bool
	log     *log.Logger
	client  http.Client // nolint: unused,deadcode
	os      string
	recordq chan model.Record
	sigq    chan os.Signal
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

	ag.recordq = make(chan model.Record, 5)
	ag.sigq = make(chan os.Signal, 2)

	signal.Notify(ag.sigq, os.Interrupt, syscall.SIGPIPE, syscall.SIGTERM)

	ag.log.Printf("[DEBUG] Agent coming up on %s, running %s %s\n",
		ag.name,
		ag.os,
		version)

	return ag, nil
} // func Create(srv string) (*Agent, error)

func (ag *Agent) readConfig(path string) error {
	var (
		fh  *os.File
		cfg config
		buf bytes.Buffer
		err error
	)

	if fh, err = os.Open(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			ag.log.Printf("[INFO] Agent configuration file %s does not exist.\n",
				common.AgentConfPath)
			return nil
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
		ag.hostID = krylib.ID(cfg.HostID)
	}

	ag.server = cfg.Server

	if cfg.Probes != nil {
		for key, interval := range cfg.Probes {
			switch key {
			case "load":
				var p Probe = &LoadProbe{}
				go ag.runProbe(p, interval)
			default:
				ag.log.Printf("[ERROR] Don't know anything about probe type %q\n",
					key)
			}
		}
	}

	return nil
} // func (ag *Agent) readConfig() error

func (ag *Agent) writeConfig() error {
	var (
		err error
		cfg config
		buf []byte
		fh  *os.File
	)

	cfg = config{
		Server: ag.server,
		HostID: int64(ag.hostID),
	}

	if buf, err = json.Marshal(&cfg); err != nil {
		ag.log.Printf("[ERROR] Failed to serialize config: %s\n",
			err.Error())
		return err
	} else if fh, err = os.Create(common.AgentConfPath); err != nil {
		ag.log.Printf("[ERROR] Failed to open agent config file at %s: %s\n",
			common.AgentConfPath,
			err.Error())
		return err
	}

	defer fh.Close()

	if _, err = fh.Write(buf); err != nil {
		ag.log.Printf("[ERROR] Cannot open agent config at %s for writing: %s\n",
			common.AgentConfPath,
			err.Error())
		return err
	}

	return nil
} // func (ag *Agent) writeConfig() error

// Run executes the Agent's main loop.
func (ag *Agent) Run() {
	ag.log.Printf("[INFO] Agent starting up.\n")
	ag.active.Store(true)
	defer ag.active.Store(false)

	var (
		err    error
		rec    model.Record
		sig    os.Signal
		ticker *time.Ticker
	)

	if ag.hostID == 0 {
		if err = ag.register(); err != nil {
			ag.log.Printf("[ERROR] Failed to register with server %s: %s\n",
				ag.server,
				err.Error())
			return
		}
	}

	ticker = time.NewTicker(heartbeat)
	defer ticker.Stop()

	for ag.active.Load() {
		select {
		case <-ticker.C:
			continue
		case rec = <-ag.recordq:
			// Do something
			if err = ag.reportRecord(&rec); err != nil {
				ag.log.Printf("[ERROR] Failed to report Record to server: %s\n",
					err.Error())
			}
		case sig = <-ag.sigq:
			ag.log.Printf("[INFO] Received Signal %s, quitting Agent loop.\n",
				sig)
			return
		}
	}
} // func (ag *Agent) Run()

func (ag *Agent) register() error {
	const endpoint = "/ws/register"

	var (
		err        error
		msg        string
		serialized []byte
		addr       = fmt.Sprintf("http://%s%s",
			ag.server,
			endpoint)
		host = model.Host{
			Name: ag.name,
			OS:   ag.os,
		}
		req   *http.Request
		res   *http.Response
		reply model.Response
		buf   bytes.Buffer
		id    int64
	)

	if serialized, err = json.Marshal(&host); err != nil {
		ag.log.Printf("[ERROR] Failed to serialize Host payload: %s\n",
			err.Error())
		return err
	} else if req, err = http.NewRequest("POST", addr, bytes.NewBuffer(serialized)); err != nil {
		ag.log.Printf("[ERROR] Failed to create HTTP request to for %s: %s\n",
			addr,
			err.Error())
		return err
	} else if res, err = ag.client.Do(req); err != nil {
		ag.log.Printf("[ERROR] Failed to perform HTTP request for %s: %s\n",
			addr,
			err.Error())
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		msg = fmt.Sprintf("Server responded with Status %s",
			res.Status)
		ag.log.Printf("[ERROR] %s\n", msg)
		return errors.New(msg)
	} else if _, err = io.Copy(&buf, res.Body); err != nil {
		ag.log.Printf("[ERROR] Failed to read Response Body: %s\n",
			err.Error())
		return err
	} else if err = json.Unmarshal(buf.Bytes(), &reply); err != nil {
		ag.log.Printf("[ERROR] Cannot decode response body: %s\n\n%s\n",
			err.Error(),
			buf.Bytes())
		return err
	} else if !reply.Status {
		ag.log.Printf("[ERROR] Response status says no: %s\n",
			reply.Message)
		return errors.New(reply.Message)
	} else if id, err = strconv.ParseInt(reply.Message, 10, 64); err != nil {
		ag.log.Printf("[ERROR] Failed to decode Host ID assigned to us by the server: %s\n ==> %s\n",
			err.Error(),
			reply.Message)
		return err
	}

	ag.hostID = krylib.ID(id)

	// I should write the config file at this point.
	if err = ag.writeConfig(); err != nil {
		ag.log.Printf("[ERROR] Cannot save config to %s: %s\n",
			common.AgentConfPath,
			err.Error())
		return err
	}

	return nil
} // func (ag *Agent) register() error

func (ag *Agent) reportRecord(rec *model.Record) error {
	const endpoint = "/ws/report"
	var (
		err        error
		msg        string
		serialized []byte
		addr       = fmt.Sprintf("http://%s%s",
			ag.server,
			endpoint)
		req   *http.Request
		res   *http.Response
		reply model.Response
		buf   *bytes.Buffer
	)

	if rec == nil {
		ag.log.Printf("[ERROR] record is nil\n")
		return errors.New("record is nil")
	}

	rec.HostID = int64(ag.hostID)

	if serialized, err = json.Marshal(rec); err != nil {
		ag.log.Printf("[ERROR] Failed to serialize record: %s\n",
			err.Error())
		return err
	}

	buf = bytes.NewBuffer(serialized)

	if req, err = http.NewRequest("POST", addr, buf); err != nil {
		ag.log.Printf("[ERROR] Failed to create HTTP request to for %s: %s\n",
			addr,
			err.Error())
		return err
	} else if res, err = ag.client.Do(req); err != nil {
		ag.log.Printf("[ERROR] Failed to perform HTTP request for %s: %s\n",
			addr,
			err.Error())
		return err
	}

	defer res.Body.Close()
	buf.Reset()

	if res.StatusCode != 200 {
		msg = fmt.Sprintf("Server responded with Status %s",
			res.Status)
		ag.log.Printf("[ERROR] %s\n", msg)
		return errors.New(msg)
	} else if _, err = io.Copy(buf, res.Body); err != nil {
		ag.log.Printf("[ERROR] Failed to read Response Body: %s\n",
			err.Error())
		return err
	} else if err = json.Unmarshal(buf.Bytes(), &reply); err != nil {
		ag.log.Printf("[ERROR] Cannot decode response body: %s\n\n%s\n",
			err.Error(),
			buf.Bytes())
		return err
	} else if !reply.Status {
		ag.log.Printf("[ERROR] Response status says no: %s\n",
			reply.Message)
		return errors.New(reply.Message)
	}

	return nil
} // func (ag *Agent) reportRecord(rec *model.Record) error

func (ag *Agent) runProbe(p Probe, interval int) {
	var ticker = time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()

	var (
		err error
		rec *model.Record
	)

	for ag.active.Load() {
		<-ticker.C
		if rec, err = p.Collect(); err != nil {
			ag.log.Printf("[ERROR] Failed to get Record from Probe: %s\n",
				err.Error())
		}
		ag.recordq <- *rec
	}
} // func (ag *Agent) runProbe(p Probe)
