// Package grafana provides an input to the badgedata library to retrieve
// dashboard download count from the public Grafana API.
package grafana

import (
	"net/http"
	"strings"

	"golift.io/badgedata"
)

//nolint:gochecknoinits // This is how the plugin works.
func init() {
	dashboardInit()
	badgedata.Register("grafana", ServeHTTP)
}

// ServeHTTP is our main traffic handler.
func ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	splitPaths := strings.Split(req.URL.Path, "/")
	if len(splitPaths) < 4 {
		http.Error(resp, "missing path segments", http.StatusNotFound)
		return
	}

	switch splitPaths[3] {
	case "dashboard-count", "dashboard-counts", "dashboard-download", "dashboard-downloads":
		WriteDashboardDownloadCount(resp, req)
	default:
		http.Error(resp, "not found", http.StatusGone)
	}
}
