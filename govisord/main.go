// Copyright 2016 The Govisor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command govisord implements a daemon that can manage proceses from
// manifest files using Govisor.
//
// The flags are
//
//	-a <address>	- select the listen address, default is
//			  http://localhost:8321
//	-d <dir>	- select the directory.  manifests live in
//			  the directory "services" underneath this
//	-p <passwd>	- use Basic Auth with a password of user:bcrypt
//			  pairs.  Bcrypt is an encrypted password.
//	-g <user:pass>	- generate & use encrypted password & user
//	-e <bool>	- enable/disable (true/false) all services (true)
//	-n <name>	- name this instance, e.g. for Realm, etc.
//
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gdamore/govisor"
	"github.com/gdamore/govisor/server"
)

var addr string = "http://127.0.0.1:8321"

type MyHandler struct {
	h      *server.Handler
	auth   bool
	passwd map[string]string
	name   string
}

func (h *MyHandler) needAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate",
		fmt.Sprintf("Basic realm=%q", h.name))
	http.Error(w, http.StatusText(http.StatusUnauthorized),
		http.StatusUnauthorized)
}

func (h *MyHandler) loadPasswdFile(name string) error {
	file, e := os.Open(name)
	if e != nil {
		return e
	}
	rd := csv.NewReader(file)
	rd.Comment = '#'
	rd.Comma = ':'
	rd.FieldsPerRecord = 2
	for {
		rec, e := rd.Read()
		if e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		h.passwd[rec[0]] = rec[1]
	}
	h.auth = true
	file.Close()
	return nil
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Consider adding logging, and timeouts, to mitigate
	if h.auth {
		user, pass, ok := r.BasicAuth()
		if !ok {
			h.needAuth(w, r)
			return
		}
		enc, ok := h.passwd[user]
		if !ok {
			h.needAuth(w, r)
			return
		}
		if e := bcrypt.CompareHashAndPassword([]byte(enc), []byte(pass)); e != nil {
			h.needAuth(w, r)
			return
		}
	}
	h.h.ServeHTTP(w, r)
}

var logFile = ""

func die(format string, v ...interface{}) {
	if logFile != "" {
		log.Printf(format, v...)
	}
	fmt.Printf(format+"\n", v...)
	os.Exit(1)
}

func main() {
	dir := "."
	name := "govisord"
	enable := true
	passFile := ""
	genpass := ""
	certFile := ""
	keyFile := ""
	m := govisor.NewManager(name)

	flag.StringVar(&certFile, "certfile", certFile, "certificate file (for TLS)")
	flag.StringVar(&keyFile, "keyfile", keyFile, "key file (for TLS)")
	flag.StringVar(&addr, "addr", addr, "listen address")
	flag.StringVar(&dir, "dir", dir, "configuration directory")
	flag.StringVar(&name, "name", name, "govisor name")
	flag.BoolVar(&enable, "enable", enable, "enable all services")
	flag.StringVar(&passFile, "passfile", passFile, "password file")
	flag.StringVar(&genpass, "passwd", genpass, "generate password")
	flag.StringVar(&logFile, "logfile", logFile, "log file")
	flag.Parse()

	var lf *os.File
	var e error
	if logFile != "" {
		lf, e = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if e != nil {
			die("Failed to open log file: %v", e)
		}
		log.SetOutput(lf)
		m.SetLogger(log.New(lf, "", log.LstdFlags))
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	h := &MyHandler{
		h:      server.NewHandler(m),
		name:   name,
		auth:   false,
		passwd: make(map[string]string),
	}
	if genpass != "" {
		h.auth = true
		rec := strings.SplitN(genpass, ":", 2)
		if len(rec) != 2 {
			die("Missing user:password")
		}
		enc, e := bcrypt.GenerateFromPassword([]byte(rec[1]), 0)
		if e != nil {
			die("bcrypt: %v", e)
		}
		h.passwd[rec[0]] = string(enc)
		log.Printf("Encrypted password is '%s'", string(enc))
	}
	if passFile != "" {
		if e := h.loadPasswdFile(passFile); e != nil {
			die("Unable to load passwd file: %v", e)
		}
	} else if _, err := os.Stat(path.Join(dir, "passwd")); err == nil {
		if e := h.loadPasswdFile(path.Join(dir, "passwd")); e != nil {
			die("Unable to load passwd file: %v", e)
		}
	}

	if certFile == "" {
		certFile = path.Join(dir, "cert.pem")
	}
	if keyFile == "" {
		keyFile = path.Join(dir, "key.pem")
	}

	go func() {
		var e error
		if strings.HasPrefix(addr, "https://") {
			e = http.ListenAndServeTLS(addr[len("https://"):],
				certFile, keyFile, h)
		} else if strings.HasPrefix(addr, "http://") {
			e = http.ListenAndServe(addr[len("http://"):], h)
		} else {
			e = http.ListenAndServe(addr, h)
		}
		if e != nil {
			die("HTTP/HTTPS failed: %v", e)
		}
	}()

	/* This sleep is long enough to verify that our HTTP service started */
	time.Sleep(time.Millisecond * 100)

	svcDir := path.Join(dir, "services")
	if d, e := os.Open(svcDir); e != nil {
		die("Failed to open services directory %s: %v", svcDir, e)
	} else if files, e := d.Readdirnames(-1); e != nil {
		die("Failed to scan scan services: %v", e)
	} else {
		for _, f := range files {
			fname := path.Join(svcDir, f)
			if mf, e := os.Open(fname); e != nil {
				log.Printf("Failed to open manifest %s: %v",
					fname, e)
			} else if p, e := govisor.NewProcessFromJson(mf); e != nil {
				log.Printf("Failed to load manifest %s: %v",
					fname, e)
				mf.Close()
			} else if e := m.AddService(p); e != nil {
				/* Failure logged by m already */
				mf.Close()
			}
		}
	}

	m.StartMonitoring()
	if enable {
		svcs, _, _ := m.Services()
		for _, s := range svcs {
			s.Enable()
		}
	}

	// Set up a handler, so that we shutdown cleanly if possible.
	go func() {
		<-sigs
		done <- true
	}()

	// Wait for a termination signal, and shutdown cleanly if we get it.
	<-done
	m.Shutdown()
	os.Exit(1)
}
