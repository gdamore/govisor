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

// Command govisor implements a client application that communicate to
// govisord.  It uses subcommands.
//
// The flags are
//
//	-a <address>	- select the listen address, default is
//			  http://localhost:8321
//	-u <user:pass>	- user name & password for basic auth
//
// Subcommands are
//
//      services            - list all services
//      status [<svc> ...]  - show status for the named services (or all)
//      info <svc>          - show more detailed service info
//      enable  <svc>       - enable the named service
//      disable <svc>       - disable the named service
//      restart <svc>       - restart the named service
//      clear <svc>         - clear the named service
//      log <svc>           - obtain the log for the named service
//
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/govisor/rest"
)

var addr string = "http://127.0.0.1:8321"
var auth string = ""

func usage() {
	log.Fatalf("Usage: %s [-a <address>] [-u <user:pass>] <subcommand>",
		os.Args[0])
}

func status(s *rest.ServiceInfo) string {
	if !s.Enabled {
		return "disabled"
	}
	if s.Failed {
		return "failed"
	}
	if s.Running {
		return "running"
	}
	return "standby"
}

func showStatus(s *rest.ServiceInfo) {
	d := time.Since(s.TimeStamp)
	// for printing second resolution is sufficient
	d -= d % time.Second
	fmt.Printf("%10s %10s %10s %s\n", s.Name,
		status(s), d.String(), s.Status)
}

type sorted []*rest.ServiceInfo

func (s sorted) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sorted) Len() int {
	return len(s)
}

func (s sorted) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	if a.Failed != b.Failed {
		// put failed items at front
		return a.Failed
	}
	if a.Enabled != b.Enabled {
		// enabled in front of non-enabled items
		return a.Enabled
	}
	// We don't worry about suspended items vs. running -- no clear order
	// there.  We just sort based on name
	return a.Name < b.Name
}

func sortInfos(items []*rest.ServiceInfo) {
	sort.Sort(sorted(items))
}

func main() {
	flag.StringVar(&addr, "a", addr, "govisor address")
	flag.StringVar(&auth, "u", auth, "user:pass authentication")
	flag.Parse()

	client := rest.NewClient(nil, addr)
	if auth != "" {
		a := strings.SplitN(auth, ":", 2)
		if len(a) != 2 {
			log.Fatalf("Bad user:pass supplied")
		}
		client.SetAuth(a[0], a[1])
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"ui"}
	}

	switch args[0] {
	case "services":
		if len(args) != 1 {
			usage()
		}
		s, e := client.Services()
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}
		sort.Strings(s)
		for _, name := range s {
			fmt.Println(name)
		}
	case "enable":
		if len(args) != 2 {
			usage()
		}
		e := client.EnableService(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}
	case "disable":
		if len(args) != 2 {
			usage()
		}
		e := client.DisableService(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}

	case "restart":
		if len(args) != 2 {
			usage()
		}
		e := client.RestartService(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}

	case "clear":
		if len(args) != 2 {
			usage()
		}
		e := client.ClearService(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}

	case "log":
		if len(args) != 2 {
			usage()
		}
		s, e := client.GetServiceLog(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}
		for _, line := range s {
			fmt.Println(line)
		}
	case "info":
		if len(args) != 2 {
			usage()
		}
		s, e := client.GetService(args[1])
		if e != nil {
			log.Fatalf("Failed: %v", e)
		}
		fmt.Printf("Name:      %s\n", s.Name)
		fmt.Printf("Desc:      %s\n", s.Description)
		fmt.Printf("Status:    %s\n", status(s))
		fmt.Printf("Since:     %v\n", time.Now().Sub(s.TimeStamp))
		fmt.Printf("Detail:    %s\n", s.Status)
		fmt.Printf("Provides: ")
		for _, p := range s.Provides {
			fmt.Printf(" %s", p)
		}
		fmt.Printf("\n")
		fmt.Printf("Depends:   ")
		for _, p := range s.Depends {
			fmt.Printf(" %s", p)
		}
		fmt.Printf("\n")
		fmt.Printf("Conflicts: ")
		for _, p := range s.Conflicts {
			fmt.Printf(" %s", p)
		}
		fmt.Printf("\n")
	case "status":
		names := args[1:]
		var e error
		if len(names) == 0 {
			names, e = client.Services()
			if e != nil {
				log.Fatalf("Failed: %v", e)
			}
		}
		if len(names) == 0 {
			// No services?
			return
		}
		infos := []*rest.ServiceInfo{}
		for _, n := range names {
			info, e := client.GetService(n)
			if e == nil {
				infos = append(infos, info)
			} else {
				log.Printf("Failed: %v", e)
			}
		}
		sortInfos(infos)
		for _, info := range infos {
			showStatus(info)
		}
	case "ui":
		doUI(client, addr)
	}
}
