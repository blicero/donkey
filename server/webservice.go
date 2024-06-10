// /home/krylon/go/src/github.com/blicero/donkey/server/webservice.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-10 21:31:09 krylon>
//
// Code to handle interactions with Clients, i.e. the web service interface

package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/blicero/donkey/database"
)

func (srv *Server) handleClientReport(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s from %s\n",
		r.URL.EscapedPath(),
		r.RemoteAddr)

	var (
		err error
		db  *database.Database
		msg string
		buf bytes.Buffer
	)

	if _, err = io.Copy(&buf, r.Body); err != nil {
		res.Message = fmt.Sprintf("Failed to read HTTP request body: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto SEND_RESPONSE
	}
} // func (srv *Server) handleClientReport(w http.ResponseWriter, r *http.Request)
