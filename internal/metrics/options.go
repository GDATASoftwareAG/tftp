package metrics

import (
	"fmt"
	"net/http"
	netPProf "net/http/pprof"
	"runtime"
	runtimePProf "runtime/pprof"
)

const (
	pprofIndexPath   = "/debug/pprof/"
	pprofCmdLinePath = "/debug/pprof/cmdline"
	pprofProfilePath = "/debug/pprof/profile"
	pprofSymbolPath  = "/debug/pprof/symbol"
	pprofTracePath   = "/debug/pprof/trace"
	pprofHeapPath    = "/debug/pprof/heap"
)

type ServerOption func(mux *http.ServeMux) *http.ServeMux

func WithProfilingHooks() ServerOption {
	return func(mux *http.ServeMux) *http.ServeMux {
		mux.HandleFunc(pprofIndexPath, netPProf.Index)
		mux.HandleFunc(pprofCmdLinePath, netPProf.Cmdline)
		mux.HandleFunc(pprofProfilePath, netPProf.Profile)
		mux.HandleFunc(pprofSymbolPath, netPProf.Symbol)
		mux.HandleFunc(pprofTracePath, netPProf.Trace)
		mux.HandleFunc(pprofHeapPath, memProfileHandler)

		return mux
	}
}

func memProfileHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="profile"`)

	runtime.GC()
	if err := runtimePProf.WriteHeapProfile(w); err != nil {
		serveError(w, http.StatusInternalServerError,
			fmt.Sprintf("Could not write heap profile: %s", err))
	}
}

func serveError(w http.ResponseWriter, status int, txt string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Go-Pprof", "1")
	w.Header().Del("Content-Disposition")
	w.WriteHeader(status)
	fmt.Fprintln(w, txt)
}
