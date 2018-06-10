// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"log"
	"os"
	"time"
)

var (
	ErrorLogger = log.New(os.Stderr, "", log.LstdFlags)
)

type LogEntry struct {
	Now             time.Time
	BackendDuration time.Duration
	TotalDuration   time.Duration
	BackendKey      string
	RemoteAddr      string
	Method          string
	Path            string
	Proto           string
	Referer         string
	UserAgent       string
	RequestIDHeader string
	RequestID       string
	StatusCode      int
	ContentLength   int64
}

func LogError(location string, path string, err error) {
	ErrorLogger.Print("ERROR in ", location, " - ", path, " - ", err.Error())
}
