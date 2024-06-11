// /home/krylon/go/src/github.com/blicero/donkey/agent/util.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2024-06-11 18:42:23 krylon>

package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blicero/krylib"
)

const releaseFile = "/etc/os-release"

var linePat = regexp.MustCompile(`^(\w+)="([^"]+)"`)

// DetectOS detects the operating system name. Works by calling uname, so this does not
// work on Windows.
func DetectOS() (string, string, error) {
	var (
		err    error
		outstr string
		pieces []string
		output []byte
		cmd    = exec.Command("/usr/bin/uname", "-rs")
	)

	if output, err = cmd.Output(); err != nil {
		return "", "", err
	}

	outstr = krylib.Chomp(string(output))

	pieces = krylib.SplitOnWhitespace(outstr)

	if len(pieces) != 2 {
		return "", "",
			fmt.Errorf("Cannot parse output of uname(1): %q",
				string(outstr))
	}

	if pieces[0] == "Linux" {
		return parseOSRelease(releaseFile)
	}

	return pieces[0], pieces[1], nil
} // func DetectOS() (string, error)

// parseOSRelease attempts to extract the name and version of the system we are
// running on from /etc/os-release.
func parseOSRelease(path string) (string, string, error) {
	var (
		err                 error
		line, name, version string
		fh                  *os.File
		rdr                 *bufio.Reader
	)

	if fh, err = os.Open(path); err != nil {
		return "", "", err
	}

	defer fh.Close() // nolint: errcheck

	rdr = bufio.NewReader(fh)

	for line, err = rdr.ReadString('\n'); err == nil && line != ""; line, err = rdr.ReadString('\n') {
		var match = linePat.FindStringSubmatch(line)

		if len(match) != 3 {
			continue
		}

		var key = strings.ToLower(match[1])

		// fmt.Fprintf(os.Stderr,
		// 	"%s = %q\n",
		// 	key,
		// 	match[2])

		if key == "name" {
			name = match[2]
		} else if key == "version_id" && version == "" {
			version = match[2]
		} else if key == "version" {
			version = match[2]
		}
	}

	if err == io.EOF {
		err = nil
	}

	return name, version, err
} // func DetectOSVersion() (string, string, error)
