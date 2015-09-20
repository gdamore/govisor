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

package ui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/gdamore/topsl"

	"github.com/gdamore/govisor/govisor/util"
	"github.com/gdamore/govisor/rest"
)

type App struct {
	view      topsl.View
	panel     topsl.Widget
	info      *InfoPanel
	help      *HelpPanel
	log       *LogPanel
	main      *MainPanel
	client    *rest.Client
	logger    *log.Logger
	err       error
	items     []*rest.ServiceInfo
	selected  *rest.ServiceInfo
	logName   string
	logInfo   *rest.LogInfo
	logErr    error
	logCtx    context.Context
	logCancel context.CancelFunc
}

func (a *App) show(w topsl.Widget) {
	if w != a.panel {
		a.panel.SetView(nil)
		a.panel = w
	}
	a.panel.SetView(a.view)
}

func (a *App) ShowHelp() {
	a.show(a.help)
}

func (a *App) ShowInfo(name string) {
	a.info.SetName(name)
	a.show(a.info)
}

func (a *App) ShowLog(name string) {
	if a.logCancel != nil {
		a.logCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	a.logInfo = nil
	a.logName = name
	a.logCtx = ctx
	a.logCancel = cancel
	a.log.SetName(name)
	go a.refreshLog(ctx, name)

	a.show(a.log)
}

func (a *App) ShowMain() {
	a.show(a.main)
}

func (a *App) DisableService(name string) {
	a.client.DisableService(name)
}

func (a *App) EnableService(name string) {
	a.client.EnableService(name)
}

func (a *App) ClearService(name string) {
	a.client.ClearService(name)
}

func (a *App) RestartService(name string) {
	a.client.RestartService(name)
}

func (a *App) Die(s string) {
	topsl.AppFini()
	fmt.Printf("Failure: %s", s)
	os.Exit(1)
}

func (a *App) Quit() {
	topsl.AppFini()
	os.Exit(0)
}

func (a *App) SetLogger(logger *log.Logger) {
	a.logger = logger
	if logger != nil {
		logger.Printf("Start logger")
	}
}

func (a *App) Logf(fmt string, v ...interface{}) {
	if a.logger != nil {
		a.logger.Printf(fmt, v...)
	}
}

func (a *App) HandleEvent(ev topsl.Event) bool {
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			// We intercept a few control keys up front, for global
			// handling.
			switch ev.Key {
			case topsl.KeyCtrlC:
				a.Quit()
				return true
			case topsl.KeyCtrlL:
				topsl.AppRedraw()
				return true
			}
		}
	}

	if a.panel != nil {
		return a.panel.HandleEvent(ev)
	}
	return false
}

func (a *App) Draw() {
	if a.panel != nil {
		a.panel.Draw()
	}
}

func (a *App) Resize() {
	if a.panel != nil {
		a.panel.Resize()
	}
}

func (a *App) SetView(view topsl.View) {
	a.view = view
	if a.panel != nil {
		a.panel.SetView(view)
	}
}

func (a *App) GetClient() *rest.Client {
	return a.client
}

func (a *App) GetAppName() string {
	return "Govisor v0.1"
}

func NewApp(client *rest.Client, url string) *App {

	app := &App{}
	app.client = client
	app.info = NewInfoPanel(app)
	app.help = NewHelpPanel(app)
	app.log = NewLogPanel(app)
	app.main = NewMainPanel(app, url)
	app.panel = app.main

	go app.refresh()
	return app
}

// refresh keeps the app items current

func (a *App) getItems() ([]*rest.ServiceInfo, error) {
	names, e := a.client.Services()
	if e != nil {
		return nil, e
	}
	items := make([]*rest.ServiceInfo, 0, len(names))
	for _, n := range names {
		item, e := a.client.GetService(n)
		if e == nil {
			items = append(items, item)
		}
	}
	util.SortServices(items)
	return items, nil
}

func (a *App) refresh() {
	client := a.client
	etag := ""
	for {
		items, e := a.getItems()

		topsl.AppLock()
		a.items = items
		a.err = e
		topsl.AppUnlock()
		topsl.AppDraw()
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Hour)
		etag, e = client.Watch(ctx, etag)
		cancel()
		if e != nil {
			time.Sleep(2 * time.Second)
		}
	}
}

func (a *App) refreshLog(ctx context.Context, name string) {
	info, e := a.client.GetLog(name)

	for {
		topsl.AppLock()
		if a.logName == name {
			a.logInfo = info
			a.logErr = e

			topsl.AppUnlock()
			topsl.AppDraw()
		} else {
			topsl.AppUnlock()
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		info, e = a.client.WatchLog(ctx, name, info)
	}
}

// Must be called with AppLock held
func (a *App) GetItems() ([]*rest.ServiceInfo, error) {
	return a.items, a.err
}

func (a *App) GetItem(name string) (*rest.ServiceInfo, error) {
	if a.err != nil {
		return nil, a.err
	}
	for _, i := range a.items {
		if i.Name == name {
			return i, nil
		}
	}
	return nil, errors.New("Service not found")
}

func (a *App) GetLog(name string) (*rest.LogInfo, error) {
	if a.logName == name {
		return a.logInfo, a.logErr
	}
	return nil, nil
}
