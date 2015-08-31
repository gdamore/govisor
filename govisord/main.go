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

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/gdamore/govisor"
	"github.com/gdamore/govisor/rpc"
)

var addr string = "127.0.0.1:8321"
var dir string = "."
var name string = "govisord"
var enable bool = true

// XXX: When the daemon dies or is terminated/shutdown, we should shut down
// all child processes via m.Shutdown()

func main() {
	flag.StringVar(&addr, "a", addr, "listen address")
	flag.StringVar(&dir, "d", dir, "manifest directory")
	flag.StringVar(&name, "n", name, "govisor name")
	flag.BoolVar(&enable, "e", enable, "enable all services")
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
		for _, s := range m.Services() {
			s.Enable()
		}
	}
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		log.Fatal(http.ListenAndServe(addr, rpc.NewHandler(m)))
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
