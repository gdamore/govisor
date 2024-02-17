## govisor

[![Linux](https://img.shields.io/github/actions/workflow/status/gdamore/govisor/linux.yml?branch=main&logoColor=grey&logo=linux&label=)](https://github.com/gdamore/govisor/actions/workflows/linux.yml)
[![Windows](https://img.shields.io/github/actions/workflow/status/gdamore/govisor/windows.yml?branch=main&logoColor=grey&logo=windows&label=)](https://github.com/gdamore/govisor/actions/workflows/windows.yml)
[![GitHub License](https://img.shields.io/github/license/gdamore/govisor.svg)](https://github.com/gdamore/govisor/blob/master/LICENSE)
[![Issues](https://img.shields.io/github/issues/gdamore/govisor.svg)](https://github.com/gdamore/govisor/issues)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/gdamore/govisor)

Govisor is a framework for managing services.  It supports dependency
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

### Commercial Support

Govisor is absolutely free, but support is available if needed:

- [TideLift](https://tidelift.com/) subscriptions include support for _govisor_, as well as many other open source packages.
- [Staysail Systems Inc.](mailto:info@staysail.tech) offers direct support, and custom development around _govisor_ on an hourly basis.
