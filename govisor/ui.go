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
	"log"
	"time"

	"github.com/gdamore/govisor/rest"
	"github.com/gdamore/topsl"
)

func doUI(client *rest.Client, url string, logger *log.Logger) {
	app := NewApp(client, url)
	app.SetLogger(logger)
	app.Logf("Starting up user interface")

	topsl.AppInit()
	topsl.SetApplication(app)
	app.ShowMain()
	// periodic updates please
	go func() {
		for {
			topsl.AppDraw()
			time.Sleep(time.Second)
		}
	}()
	topsl.RunApplication()

}

/*
   Our screen has the following appearance:

    Server: http://localhost:8321/
    xxx Services  xxx Running  yyy Faulted  zzz Standby            Govisor v1.0
   ____________________________________________________________________________
   ...
   testservice:name        faulted      4d10m32s    Failed: Terminated
   ...
   testservice:ok          running            5s    Service started
   ...
   dontrunme:ever          disabled    132d10m5s    Service disabled
   ...
   ____________________________________________________________________________
   [Q]uit [I]Info [E]nable [D]isable [R]estart [C]lear [L]og  [H]elp
*/
