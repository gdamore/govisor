// Copyright 2016 The Govisor Authors
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
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"

	"github.com/gdamore/govisor/govisor/util"
	"github.com/gdamore/govisor/rest"
)

type App struct {
	app       *views.Application
	view      views.View
	panel     views.Widget
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

	views.WidgetWatchers
}

func (a *App) show(w views.Widget) {
	if w != a.panel {
		a.panel.SetView(nil)
		a.panel = w
	}
	a.panel.SetView(a.view)
	a.panel.Resize()
	a.app.Refresh()
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

func (a *App) Quit() {
	/* This just posts the quit event. */
	a.app.Quit()
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

func (a *App) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		// Intercept a few control keys up front, for global handling.
		case tcell.KeyCtrlC:
			a.Quit()
			return true
		case tcell.KeyCtrlL:
			a.app.Refresh()
			return true
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

func (a *App) SetView(view views.View) {
	a.view = view
	if a.panel != nil {
		a.panel.SetView(view)
	}
}

func (a *App) Size() (int, int) {
	if a.panel != nil {
		return a.panel.Size()
	}
	return 0, 0
}

func (a *App) GetClient() *rest.Client {
	return a.client
}

func (a *App) GetAppName() string {
	return "Govisor v1.1"
}

func NewApp(client *rest.Client, url string) *App {

	app := &App{}
	app.app = &views.Application{}
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

		a.app.PostFunc(func() {
			a.items = items
			a.err = e
			a.app.Update()
		})
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
		a.app.PostFunc(func() {
			if a.logName == name {
				a.logInfo = info
				a.logErr = e
				a.app.Update()
			}
		})
		select {
		case <-ctx.Done():
			return
		default:
		}
		info, e = a.client.WatchLog(ctx, name, info)
	}
}

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

func (a *App) Run() {
	a.Logf("Starting up user interface")
	a.app.SetRootWidget(a)
	a.ShowMain()
	go func() {
		// Give us periodic updates
		for {
			a.app.Update()
			time.Sleep(time.Second)
		}
	}()
	a.Logf("Starting app loop")
	a.app.Run()
}
