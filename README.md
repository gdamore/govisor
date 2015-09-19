## govisor

[![Linux Status](https://img.shields.io/travis/gdamore/govisor.svg?label=linux)](https://travis-ci.org/gdamore/govisor)
[![Windows Status](https://img.shields.io/appveyor/ci/gdamore/govisor.svg?label=windows)](https://ci.appveyor.com/project/gdamore/govisor)
[![GitHub License](https://img.shields.io/github/license/gdamore/govisor.svg)](https://github.com/gdamore/govisor/blob/master/LICENSE)
[![Issues](https://img.shields.io/github/issues/gdamore/govisor.svg)](https://github.com/gdamore/govisor/issues)
[![Gitter](https://img.shields.io/badge/gitter-join-brightgreen.svg)](https://gitter.im/gdamore/govisor)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/gdamore/govisor)

> _Govisor is a work in progress (Alpha).
> Please use with caution; at this
> time it is not suitable for production use._

package govisor is an framework for managing services.  It supports dependency
graphs of services, and handles starting, stopping, and restarting services
as necessary.  It also deals with failures, and supports self-healing, and
has some advanced logging capabilities.  It also offers a REST API for
managing your services.

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

