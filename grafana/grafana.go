// Package grafana provides an input to the badgedata library to retreive
// dashboard download count from the public Grafana API.
package grafana

import (
	"net/http"
	"strings"

	"golift.io/badgedata"
)

func init() {
	dashboardInit()
	badgedata.Register("grafana", ServeHTTP)
}

// ServeHTTP is our main traffic handler.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	splitPaths := strings.Split(r.URL.Path, "/")
	if len(splitPaths) < 4 {
		http.Error(w, "missing path segments", http.StatusNotFound)
		return
	}
	switch splitPaths[3] {
	case "dashboard-count", "dashboard-counts", "dashboard-download", "dashboard-downloads":
		WriteDashboardDownloadCount(w, r)
		return
		// print some cool json.
	default:
		http.Error(w, "not found", http.StatusGone)
		return
	}
}
