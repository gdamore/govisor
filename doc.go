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

// NOTE: THIS DESCRIBES INTENT, AND DOES NOT REFLECT ACTUAL IMPLEMENTATION
// (YET!!)
//
// Package govisor provides a pure Go process management framework.
// This is similar in concept to supervisord, but the implementation
// and the interfaces are wholly different.  Some inspiration is taken
// from Solaris' SMF facility.
//
// Unlike other frameworks, the intention is that this framework is not
// a replacement for your system's master process management (i.e. init,
// upstart, or similar), but rather a tool for user's (or administrators)
// to manage their own groups of processes as part of application
// deployment.
//
// Multiple instances of govisor may be deployed, and an instance may
// be deployed using Go's HTTP handler framework, so that it is possible
// to register the manager within an existing server instance.
//
package govisor
