// Package badgedata provides a collection of methods to retrieve, store and re-display
// data from other websites. The intent is to use the displayed data with badgen.net.
package badgedata

import (
	"net/http"
	"strings"
	"sync"
)

//nolint:gochecknoglobals
var (
	routersMu sync.Mutex
	routes    routers
)

type routers map[string]http.HandlerFunc

// Handler returns the main handler for /badgedata endpoint.
func Handler() http.HandlerFunc {
	routersMu.Lock()
	defer routersMu.Unlock()

	// We copy all the routes into a new map so we can avoid locking on every request.
	reroute := make(routers)
	for i, v := range routes {
		reroute[i] = v
	}

	return reroute.ServeHTTP
}

func (routeMap routers) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")
	if len(path) < 3 {
		http.Error(resp, "missing path segments", http.StatusNotFound)
		return
	}
	route := path[2]
	handler, ok := routeMap[route]
	if !ok {
		http.Error(resp, "not found: "+route, http.StatusNotFound)
		return
	}

	handler.ServeHTTP(resp, req)
}

// Register should only be called from init functions.
// Registrations created after calling Handler() will not work.
func Register(name string, function http.HandlerFunc) {
	routersMu.Lock()
	defer routersMu.Unlock()
	if routes == nil {
		routes = make(routers)
	}
	routes[name] = function
}
