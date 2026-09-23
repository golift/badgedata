package grafana //nolint:testpackage // tests replace DashboardAPI and the cache.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestServeHTTPRejectsBadRoutes(t *testing.T) {
	t.Parallel()

	short := serve(t, "/badgedata/grafana")
	if short.Code != http.StatusNotFound {
		t.Fatalf("short path status %d", short.Code)
	}

	unknown := serve(t, "/badgedata/grafana/nope")
	if unknown.Code != http.StatusGone {
		t.Fatalf("unknown route status %d", unknown.Code)
	}

	tooMany := strings.TrimRight(strings.Repeat("1,", maxDashboardIDs+1), ",")
	many := serve(t, "/badgedata/grafana/dashboard-count/"+tooMany)

	if many.Code != http.StatusInternalServerError {
		t.Fatalf("too many ids status %d body %s", many.Code, many.Body.String())
	}

	badID := serve(t, "/badgedata/grafana/dashboard-count/abc")
	if badID.Code != http.StatusInternalServerError || !strings.Contains(badID.Body.String(), "invalid dashboard ID") {
		t.Fatalf("bad id status %d body %s", badID.Code, badID.Body.String())
	}
}

func TestServeHTTPDashboardCountUsesCache(t *testing.T) { //nolint:paralleltest // mutates the package dashboard cache.
	var hits int

	upstream := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		hits++

		dashID := strings.Trim(req.URL.Path, "/")

		parsedID, err := strconv.ParseInt(dashID, 10, 64)
		if err != nil {
			http.Error(resp, "bad id", http.StatusBadRequest)
			return
		}

		_, err = fmt.Fprintf(resp, `{"name":"demo","id":%d,"downloads":7}`, parsedID)
		if err != nil {
			return
		}
	}))
	t.Cleanup(upstream.Close)

	orig := DashboardAPI
	DashboardAPI = upstream.URL + "/"

	t.Cleanup(func() { DashboardAPI = orig })

	dashboarMu.Lock()
	origBoards := dashboards
	dashboards = map[string]Dashboard{}
	dashboarMu.Unlock()

	t.Cleanup(func() {
		dashboarMu.Lock()
		dashboards = origBoards
		dashboarMu.Unlock()
	})

	got := serve(t, "/badgedata/grafana/dashboard-count/42")
	if got.Code != http.StatusOK {
		t.Fatalf("dashboard status %d body %s", got.Code, got.Body.String())
	}

	if got.Body.String() != `{"subject": "1 dashboards", "status": 7}` {
		t.Fatalf("body %s", got.Body.String())
	}

	cached := serve(t, "/badgedata/grafana/dashboard-count/42")
	if cached.Body.String() != got.Body.String() {
		t.Fatalf("cached body %s", cached.Body.String())
	}

	if hits != 1 {
		t.Fatalf("upstream hits %d, want 1", hits)
	}
}

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody)
	ServeHTTP(rec, req)

	return rec
}
