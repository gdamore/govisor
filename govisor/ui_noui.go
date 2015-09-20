// +build dragonfly solaris plan9 nacl

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
	"fmt"
	"log"
	"os"

	"github.com/gdamore/govisor/rest"
)

func doUI(client *rest.Client, url string, logger *log.Logger) {

	fmt.Fprintf(os.Stderr,
		"TOPSL based UI not available on this platform.\n")
	os.Exit(1)
}
