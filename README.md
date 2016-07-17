## govisor

[![Linux Status](https://img.shields.io/travis/gdamore/govisor.svg?label=linux)](https://travis-ci.org/gdamore/govisor)
[![Windows Status](https://img.shields.io/appveyor/ci/gdamore/govisor.svg?label=windows)](https://ci.appveyor.com/project/gdamore/govisor)
[![GitHub License](https://img.shields.io/github/license/gdamore/govisor.svg)](https://github.com/gdamore/govisor/blob/master/LICENSE)
[![Issues](https://img.shields.io/github/issues/gdamore/govisor.svg)](https://github.com/gdamore/govisor/issues)
[![Gitter](https://img.shields.io/badge/gitter-join-brightgreen.svg)](https://gitter.im/gdamore/govisor)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/gdamore/govisor)

package govisor is a framework for managing services.  It supports dependency
graphs of services, and handles starting, stopping, and restarting services
as necessary.  It also deals with failures, and supports self-healing, and
has some advanced logging capabilities.  It also offers a REST API for
managing your services, as well as a nicer client API, and a snazzy little
terminal application to monitor the services.

There is a daemon (govisord) that can be used to manage a tree of process in
a manner similar to init or SMF or systemd.  However, it is designed to be
suitable for use by unprivileged users, and it is possible to run many copies
of govisord on the same system (but you will have to choose different TCP
management ports.)

Govisord listens by default at http://localhost:8321/

See govisord/main.go for a list of options, etc.

Govisor is designed for embedding as well.  You can embed the manager into
your own application.  The REST API implementation provides a http.Handler,
so you can also wrap or embed the API with other web services.

The govisor client application, "govisor", is in the govisor/ directory.
It has a number of options, try it with -h to see them.

To install the daemon and client: `go get -v github.com/gdamore/govisor/...`
