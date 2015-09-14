// Copyright 2015 The Govisor Authors
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

	"golang.org/x/crypto/bcrypt"

	"github.com/gdamore/govisor"
	"github.com/gdamore/govisor/rest"
)

var addr string = "127.0.0.1:8321"
var dir string = "."
var name string = "govisord"
var enable bool = true
var passfile string = ""
var genpass string = ""

type MyHandler struct {
	h      *rest.Handler
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
		h.passwd[rec[0]] = h.passwd[rec[1]]
	}
	file.Close()
	return nil
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.auth {
		if user, pass, ok := r.BasicAuth(); !ok {
			h.needAuth(w, r)
			return
		} else if enc, ok := h.passwd[user]; !ok {
			h.needAuth(w, r)
			return
		} else if e := bcrypt.CompareHashAndPassword([]byte(enc), []byte(pass)); e != nil {
			h.needAuth(w, r)
			return
		}
	}
	h.h.ServeHTTP(w, r)
}

func main() {
	flag.StringVar(&addr, "a", addr, "listen address")
	flag.StringVar(&dir, "d", dir, "manifest directory")
	flag.StringVar(&name, "n", name, "govisor name")
	flag.BoolVar(&enable, "e", enable, "enable all services")
	flag.StringVar(&passfile, "p", passfile, "password file")
	flag.StringVar(&genpass, "g", genpass, "generate password")
	flag.Parse()

	m := govisor.NewManager(name)
	m.StartMonitoring()

	svcDir := path.Join(dir, "services")

	if d, e := os.Open(svcDir); e != nil {
		log.Fatalf("Failed to open services directory %s: %v",
			svcDir, e)
	} else if files, e := d.Readdirnames(-1); e != nil {
		log.Fatalf("Failed to scan scan services: %v", e)
	} else {
		for _, f := range files {
			fname := path.Join(svcDir, f)
			if mf, e := os.Open(fname); e != nil {
				log.Printf("Failed to open manifest %s: %v",
					fname, e)
				mf.Close()
				continue
			} else if p, e := govisor.NewProcessFromJson(mf); e != nil {
				log.Printf("Failed to load manifest %s: %v",
					fname, e)
				mf.Close()
				continue
			} else {
				m.AddService(p)
				mf.Close()
			}
		}
	}
	if enable {
		svcs, _, _ := m.Services()
		for _, s := range svcs {
			s.Enable()
		}
	}
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	h := &MyHandler{
		h:      rest.NewHandler(m),
		name:   name,
		auth:   false,
		passwd: make(map[string]string),
	}
	if genpass != "" {
		h.auth = true
		rec := strings.SplitN(genpass, ":", 2)
		if len(rec) != 2 {
			log.Fatalf("Missing user:password")
		}
		enc, e := bcrypt.GenerateFromPassword([]byte(rec[1]), 0)
		if e != nil {
			log.Fatalf("bccrypt: %v", e)
		}
		h.passwd[rec[0]] = string(enc)
		log.Printf("Encrypted password is %s\n", string(enc))
	}
	if passfile != "" {
		h.auth = true
		if e := h.loadPasswdFile(passfile); e != nil {
			log.Fatalf("Unable to load passwd file: %v", e)
		}
	}

	go func() {
		log.Fatal(http.ListenAndServe(addr, h))
	}()

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
