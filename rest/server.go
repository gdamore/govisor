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

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	svcs := h.m.Services()
	l := make([]string, 0, len(svcs))
	for _, svc := range svcs {
		l = append(l, svc.Name())
	}

	h.writeJson(w, l)
}

func (h *Handler) findService(name string) (*govisor.Service, *Error) {
	for _, svc := range h.m.Services() {
		if svc.Name() == name {
			return svc, nil
		}
	}
	return nil, &Error{http.StatusNotFound, "Service not found"}
}

func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["service"]
	if svc, e := h.findService(name); e != nil {
		h.writeError(w, e)
	} else {
		info := &ServiceInfo{
			Name:        svc.Name(),
			Description: svc.Description(),
			Enabled:     svc.Enabled(),
			Running:     svc.Running(),
			Failed:      svc.Failed(),
			Provides:    svc.Provides(),
			Depends:     svc.Depends(),
			Conflicts:   svc.Conflicts(),
		}
		info.Status, info.TimeStamp = svc.Status()
		h.writeJson(w, info)
	}
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.r.ServeHTTP(w, req)
}

func NewHandler(m *govisor.Manager) *Handler {
	r := mux.NewRouter()
	h := &Handler{m: m, r: r}
	r.HandleFunc("/services", h.listServices).Methods("GET")
	r.HandleFunc("/services/{service}", h.getService).Methods("GET")
	r.HandleFunc("/services/{service}/enable", h.enableService).Methods("POST")
	r.HandleFunc("/services/{service}/disable", h.disableService).Methods("POST")
	r.HandleFunc("/services/{service}/clear", h.clearService).Methods("POST")
	r.HandleFunc("/services/{service}/restart", h.restartService).Methods("POST")
	r.HandleFunc("/services/{service}/log", h.getLog).Methods("GET")
	return h
}
