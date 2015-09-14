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

package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gdamore/govisor"
	"github.com/gorilla/mux"
)

// Handler wraps a Manager, adding http.Handler functionality.
type Handler struct {
	m *govisor.Manager
	r *mux.Router
}

func (h *Handler) internalError(w http.ResponseWriter, e error) {
	http.Error(w, e.Error(), http.StatusInternalServerError)
}

func (h *Handler) writeJson(w http.ResponseWriter, v interface{}) {
	if b, e := json.Marshal(v); e != nil {
		h.internalError(w, e)
	} else {
		w.Header().Set("Content-Type", mimeJson)
		w.Write(b)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, e *Error) {
	if b, err := json.Marshal(e); err != nil {
		h.internalError(w, err)
	} else {
		w.Header().Set("Content-Type", mimeJson)
		w.WriteHeader(e.Code)
		w.Write(b)
	}
}

func (h *Handler) checkPoll(r *http.Request,
	watchFn func(old int64, expire time.Duration) int64) {
	if ptag := r.Header.Get(PollEtagHeader); len(ptag) < 2 {
		return
	} else if ptag[0] != '"' || ptag[len(ptag)-1] != '"' {
		return
	} else if v, e := strconv.ParseInt(ptag[1:len(ptag)-1], 16, 64); e != nil {
		return
	} else {
		ptime, _ := strconv.Atoi(r.Header.Get(PollTimeHeader))
		watchFn(v, time.Duration(ptime)*time.Second)
	}
}

func (h *Handler) condCheckGet(w http.ResponseWriter, r *http.Request,
	etag string, ts time.Time) bool {

	if chkTag := r.Header.Get("If-None-Match"); chkTag == etag {
		w.WriteHeader(http.StatusNotModified)
		return false
	}
	if chkTime := r.Header.Get("If-Modified-Since"); chkTime == "" {
		return true
	} else {
		when, e := time.Parse(http.TimeFormat, chkTime)
		if e != nil {
			return true
		}
		// Round up to 1 nearest second.  Note that the carefully
		// chosen use of ts.Before means that this will match
		// if the same timestamp is used.
		if ts.Before(when.Add(time.Second)) {
			w.WriteHeader(http.StatusNotModified)
			return false
		}
	}
	return true
}

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {

	h.checkPoll(r, h.m.WatchServices)
	svcs, sn, ts := h.m.Services()
	l := make([]string, 0, len(svcs))

	for _, svc := range svcs {
		l = append(l, svc.Name())
	}
	etag := "\"" + strconv.FormatInt(sn, 16) + "\""
	if !h.condCheckGet(w, r, etag, ts) {
		return
	}
	w.Header().Set("Etag", etag)
	w.Header().Set("Last-Modified", ts.UTC().Format(http.TimeFormat))
	h.writeJson(w, l)
}

// TODO consider conditionalizing this
func (h *Handler) findService(name string) (*govisor.Service, *Error) {
	svcs, _, _ := h.m.Services()
	for _, svc := range svcs {
		if svc.Name() == name {
			return svc, nil
		}
	}
	return nil, &Error{http.StatusNotFound, "Service not found"}
}

func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	name := vars["service"]
	svc, e := h.findService(name)
	if e != nil {
		h.writeError(w, e)
		return
	}
	h.checkPoll(r, svc.WatchService)
	var info *ServiceInfo
	// This loop ensures we provide a consistent view of
	// the service.  We assume (hope!) that the service isn't
	// changing so quickly that we can't complete all these in
	// the time it takes for a single loop iteration.
	for {
		sn := svc.Serial() // must be at start
		info = &ServiceInfo{
			Name:        svc.Name(),
			Description: svc.Description(),
			Enabled:     svc.Enabled(),
			Running:     svc.Running(),
			Failed:      svc.Failed(),
			Provides:    svc.Provides(),
			Depends:     svc.Depends(),
			Conflicts:   svc.Conflicts(),
			Serial:      strconv.FormatInt(sn, 16),
		}
		info.Status, info.TimeStamp = svc.Status()
		// check must be last
		if newsn := svc.Serial(); sn == newsn {
			break
		} else {
			sn = newsn
		}
	}

	etag := "\"" + info.Serial + "\""
	if !h.condCheckGet(w, r, etag, info.TimeStamp) {
		return
	}

	w.Header().Set("Etag", etag)
	w.Header().Set("Last-Modified",
		info.TimeStamp.UTC().Format(http.TimeFormat))
	h.writeJson(w, info)

}

func (h *Handler) enableService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else if err := svc.Enable(); err != nil {
		e = &Error{http.StatusBadRequest, err.Error()}
		h.writeError(w, e)
	} else {
		h.writeJson(w, ok)
	}
}

func (h *Handler) disableService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else if err := svc.Disable(); err != nil {
		e = &Error{http.StatusBadRequest, err.Error()}
		h.writeError(w, e)
	} else {
		h.writeJson(w, ok)
	}
}

func (h *Handler) restartService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else if err := svc.Restart(); err != nil {
		e = &Error{http.StatusBadRequest, err.Error()}
		h.writeError(w, e)
	} else {
		h.writeJson(w, ok)
	}
}

func (h *Handler) clearService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else {
		svc.Clear()
		h.writeJson(w, ok)
	}
}

func (h *Handler) getLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else {
		lines := svc.GetLog()
		h.writeJson(w, lines)
	}
}

func (h *Handler) getManager(w http.ResponseWriter, r *http.Request) {
	h.checkPoll(r, h.m.WatchSerial)
	info := h.m.GetInfo()
	sstr := strconv.FormatInt(info.Serial, 16)
	etag := "\"" + sstr + "\""
	i := &ManagerInfo{
		Name:       info.Name,
		Serial:     sstr,
		CreateTime: info.CreateTime,
		UpdateTime: info.UpdateTime,
	}
	if !h.condCheckGet(w, r, etag, info.UpdateTime) {
		return
	}
	w.Header().Set("Etag", etag)
	w.Header().Set("Last-Modified",
		i.UpdateTime.UTC().Format(http.TimeFormat))
	h.writeJson(w, i)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.r.ServeHTTP(w, req)
}

func NewHandler(m *govisor.Manager) *Handler {
	r := mux.NewRouter()
	h := &Handler{m: m, r: r}
	r.HandleFunc("/", h.getManager).Methods("GET")
	r.HandleFunc("/services", h.listServices).Methods("GET")
	r.HandleFunc("/services/{service}", h.getService).Methods("GET")
	r.HandleFunc("/services/{service}/enable", h.enableService).Methods("POST")
	r.HandleFunc("/services/{service}/disable", h.disableService).Methods("POST")
	r.HandleFunc("/services/{service}/clear", h.clearService).Methods("POST")
	r.HandleFunc("/services/{service}/restart", h.restartService).Methods("POST")
	r.HandleFunc("/services/{service}/log", h.getLog).Methods("GET")
	return h
}
