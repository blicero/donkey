// /home/krylon/go/src/github.com/blicero/pkman/backend/platform/system.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 04. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-04-20 23:13:41 krylon>

package platform

import (
	"errors"
	"regexp"
)

//go:generate stringer -type=System

// System represents an operating system
type System uint8

// These constants represents the various operating systems we aim to support.
// For systems that have derivatives, such as Debian, Ubuntu, Raspbian, or RHEL
// and Rocky Linux or Alma, we just use one constant, because the package manager
// is going to be the same.
const (
	FreeBSD System = iota
	OpenBSD
	NetBSD
	OpenSuse
	Debian
	Arch
	RedHat
)

var ErrUnknownOS = errors.New("Unknown OS")

var osPatterns = map[*regexp.Regexp]System{
	regexp.MustCompile("(?i)FreeBSD"):                   FreeBSD,
	regexp.MustCompile("(?i)OpenBSD"):                   OpenBSD,
	regexp.MustCompile("(?i)NetBSD"):                    NetBSD,
	regexp.MustCompile("(?i)Debian|Ubuntu|Raspbian"):    Debian,
	regexp.MustCompile("(?i)openSuse"):                  OpenSuse,
	regexp.MustCompile("(?i)Arch|Manjaro"):              Arch,
	regexp.MustCompile("(?i)Rocky|Fedora|OpenMandriva"): RedHat,
}

// ParseSystem attempts to parse the name of an operating system and return
// the matching System constant.
func ParseSystem(str string) (System, error) {
	for pat, system := range osPatterns {
		if pat.MatchString(str) {
			return system, nil
		}
	}

	return 0, ErrUnknownOS
} // func ParseSystem(str string) (System, error)

// Returns a slice of all System constants.
func AllSystems() []System {
	return []System{
		FreeBSD,
		OpenBSD,
		NetBSD,
		OpenSuse,
		Debian,
		Arch,
		RedHat,
	}
} // func AllSystems() []System
