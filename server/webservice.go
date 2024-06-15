// /home/krylon/go/src/github.com/blicero/donkey/server/webservice.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-15 18:09:57 krylon>
//
// Code to handle interactions with Clients, i.e. the web service interface

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/blicero/donkey/database"
	"github.com/blicero/donkey/model"
	"github.com/gorilla/mux"
)

//   URLs für Agent:
//   /ws/register                    -> handleClientRegister
//   /ws/report/load/{name:(?:\w+$)} -> handleClientReportLoad

func (srv *Server) handleClientRegister(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s from %s\n",
		r.URL.EscapedPath(),
		r.RemoteAddr)

	var (
		err          error
		db           *database.Database
		buf          bytes.Buffer
		body         []byte
		host, dbhost *model.Host
		msg          string
		res          model.Response
	)

	if _, err = io.Copy(&buf, r.Body); err != nil {
		res.Message = fmt.Sprintf("Failed to read HTTP request body: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}

	body = buf.Bytes()
	host = new(model.Host)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = json.Unmarshal(body, host); err != nil {
		msg = fmt.Sprintf("Failed to decode payload: %s", err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		res.Message = msg
		goto SEND_RESPONSE
	} else if dbhost, err = db.HostGetByName(host.Name); err != nil {
		res.Message = fmt.Sprintf("Failed to look up host %s in database: %s",
			host.Name,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto SEND_RESPONSE
	} else if dbhost != nil {
		res.Message = fmt.Sprintf("Cannot register host %s: Already exists in database (%d)",
			host.Name,
			dbhost.ID)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto SEND_RESPONSE
	} else if err = db.HostAdd(host); err != nil {
		res.Message = fmt.Sprintf("Error adding host %s to database: %s",
			host.Name,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}

	res.Status = true
	res.Message = strconv.Itoa(int(host.ID))

SEND_RESPONSE:
	res.Timestamp = time.Now()
	var rbuf []byte
	if rbuf, err = json.Marshal(&res); err != nil {
		srv.log.Printf("[ERROR] Error serializing response: %s\n",
			err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.WriteHeader(200)
	if _, err = w.Write(rbuf); err != nil {
		msg = fmt.Sprintf("Failed to send result: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
	}
} // func (srv *Server) handleClientRegister(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleClientReportData(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s from %s\n",
		r.URL.EscapedPath(),
		r.RemoteAddr)

	var (
		err     error
		db      *database.Database
		msg     string
		buf     bytes.Buffer
		res     model.Response
		payload model.Record
		host    *model.Host
		name    string
		body    []byte
	)

	if _, err = io.Copy(&buf, r.Body); err != nil {
		res.Message = fmt.Sprintf("Failed to read HTTP request body: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}

	body = buf.Bytes()

	if err = json.Unmarshal(body, &payload); err != nil {
		msg = fmt.Sprintf("Failed to decode payload: %s", err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		res.Message = msg
		goto SEND_RESPONSE
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

SEND_RESPONSE:
	res.Timestamp = time.Now()
	var rbuf []byte
	if rbuf, err = json.Marshal(&res); err != nil {
		srv.log.Printf("[ERROR] Error serializing response: %s\n",
			err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.WriteHeader(200)
	if _, err = w.Write(rbuf); err != nil {
		msg = fmt.Sprintf("Failed to send result: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
	}
} // func (srv *Server) handleClientReportData(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleClientReportLoad(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s from %s\n",
		r.URL.EscapedPath(),
		r.RemoteAddr)

	var (
		err     error
		db      *database.Database
		msg     string
		buf     bytes.Buffer
		res     model.Response
		payload model.Load
		host    *model.Host
		name    string
		body    []byte
	)

	vars := mux.Vars(r)

	name = vars["name"]

	if _, err = io.Copy(&buf, r.Body); err != nil {
		res.Message = fmt.Sprintf("Failed to read HTTP request body: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}

	body = buf.Bytes()

	if err = json.Unmarshal(body, &payload); err != nil {
		msg = fmt.Sprintf("Failed to decode payload: %s", err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		res.Message = msg
		goto SEND_RESPONSE
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if host, err = db.HostGetByName(name); err != nil {
		msg = fmt.Sprintf("Cannot find Host %s in database: %s",
			name,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		res.Message = msg
		goto SEND_RESPONSE
	} else if host == nil {
		res.Message = fmt.Sprintf("Did not find Host %s in database", name)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto SEND_RESPONSE
	} else if host.ID != payload.HostID {
		res.Message = fmt.Sprintf("Mismatched Host ID: Payload says %d, database says %d",
			payload.HostID,
			host.ID)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto SEND_RESPONSE
	} else if err = db.LoadAdd(&payload); err != nil {
		res.Message = fmt.Sprintf("Error adding Load for Host %d (%s) to database: %s",
			host.ID,
			host.Name,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}

	res.Status = true

SEND_RESPONSE:
	res.Timestamp = time.Now()
	var rbuf []byte
	if rbuf, err = json.Marshal(&res); err != nil {
		srv.log.Printf("[ERROR] Error serializing response: %s\n",
			err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.WriteHeader(200)
	if _, err = w.Write(rbuf); err != nil {
		msg = fmt.Sprintf("Failed to send result: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
	}
} // func (srv *Server) handleClientReportLoad(w http.ResponseWriter, r *http.Request)
