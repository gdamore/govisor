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
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gdamore/govisor/rest"
	"github.com/gdamore/govisor/govisor/util"
)

var addr string = "http://127.0.0.1:8321"

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s [flags] <subcommand> [args]", os.Args[0])
	os.Exit(1)
}

func showStatus(s *rest.ServiceInfo) {
	d := time.Since(s.TimeStamp)
	// for printing second resolution is sufficient
	d -= d % time.Second
	fmt.Printf("%-20s %-10s  %10s    %s\n", s.Name,
		util.Status(s), util.FormatDuration(d), s.Status)
}

func loadCertPath(roots *x509.CertPool, dirname string) error {
	return filepath.Walk(dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if err := loadCertFile(roots, path); err != nil {
					return err
				}
			}
			return nil
		})
}

func loadCertFile(roots *x509.CertPool, fname string) error {
	data, err := ioutil.ReadFile(fname)
	if err == nil {
		roots.AppendCertsFromPEM(data)
	}
	return err
}

func fatal(f string, e error) {
	msg := e.Error()
	if len(msg) > 4 && msg[0] == '4' {
		fmt.Fprintf(os.Stderr, "%s: %s\n", f, msg[4:])
	} else {
		fmt.Fprintf(os.Stderr, "%s: %s\n", f, msg)
	}
	os.Exit(1)
}

func main() {
	user := ""
	pass := ""
	logfile := ""
	cafile := ""
	capath := ""
	insecure := false

	flag.StringVar(&addr, "addr", addr, "govisor address")
	flag.StringVar(&user, "user", user, "user name for authentication")
	flag.StringVar(&pass, "pass", pass, "password for authentication")
	flag.StringVar(&cafile, "cacert", cafile, "CA certificate file")
	flag.StringVar(&capath, "capath", capath, "CA certificates directory")
	flag.StringVar(&logfile, "debuglog", logfile, "debug log file")
	flag.BoolVar(&insecure, "insecure", insecure, "allow insecure TLS connections")
	flag.Parse()

	var dlog *log.Logger
	if logfile != "" {
		f, e := os.Create(logfile)
		if e == nil {
			dlog = log.New(f, "DEBUG:", log.LstdFlags)
			log.SetOutput(f)
		}
	}

	roots := x509.NewCertPool()
	if cafile == "" && capath == "" {
		roots = nil
	} else {
		if cafile != "" {
			if e := loadCertFile(roots, cafile); e != nil {
				fatal("Unable to load cert file", e)
			}
		}
		if capath != "" {
			if e := loadCertPath(roots, capath); e != nil {
				fatal("Unable to load cert path", e)
			}
		}
	}

	u, e := url.Parse(addr)
	if e != nil {
		fatal("Bad address", e)
	}
	tcfg := &tls.Config{
		RootCAs:            roots,
		ServerName:         u.Host,
		InsecureSkipVerify: insecure,
	}
	tran := &http.Transport{
		TLSClientConfig: tcfg,
	}

	client := rest.NewClient(tran, addr)
	if user != "" || pass != "" {
		client.SetAuth(user, pass)
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
			fatal("Error", e)
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
			fatal("Error", e)
		}
	case "disable":
		if len(args) != 2 {
			usage()
		}
		e := client.DisableService(args[1])
		if e != nil {
			fatal("Error", e)
		}

	case "restart":
		if len(args) != 2 {
			usage()
		}
		e := client.RestartService(args[1])
		if e != nil {
			fatal("Error", e)
		}

	case "clear":
		if len(args) != 2 {
			usage()
		}
		e := client.ClearService(args[1])
		if e != nil {
			fatal("Error", e)
		}

	case "log":
		if len(args) != 2 {
			usage()
		}
		loginfo, e := client.GetLog(args[1])
		if e != nil {
			fatal("Error", e)
		}
		for _, line := range loginfo.Records {
			fmt.Printf("%s %s\n",
				line.Time.Format(time.StampMilli), line.Text)
		}
	case "info":
		if len(args) != 2 {
			usage()
		}
		s, e := client.GetService(args[1])
		if e != nil {
			fatal("Failed", e)
		}
		fmt.Printf("Name:      %s\n", s.Name)
		fmt.Printf("Desc:      %s\n", s.Description)
		fmt.Printf("Status:    %s\n", util.Status(s))
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
				fatal("Error", e)
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
				fatal("Error", e)
			}
		}
		util.SortServices(infos)
		for _, info := range infos {
			showStatus(info)
		}
	case "ui":
		doUI(client, addr, dlog)
	}
}
