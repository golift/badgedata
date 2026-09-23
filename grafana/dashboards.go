package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DashboardAPI is the URL to the JSON API at Grafana.com.
// Tests point this at a local server.
//
//nolint:gochecknoglobals // tests replace this with a local server.
var DashboardAPI = "https://grafana.com/api/dashboards/"

const (
	refreshTime           = time.Hour
	requestTimeout        = 10 * time.Second
	dashboardPathSegments = 5 // /badgedata/grafana/dashboard-count/<ids>
	maxDashboardIDs       = 50
)

// Dashboard holds a dashboard's name and download count.
// This is a small snippet of the data available from the Grafana API.
type Dashboard struct {
	Name      string    `json:"name"`
	ID        int64     `json:"id"`
	Downloads int64     `json:"downloads"`
	Time      time.Time `json:"-"`
}

//nolint:gochecknoglobals // This is the cache.
var (
	dashboards map[string]Dashboard
	dashboarMu sync.RWMutex
)

func dashboardInit() {
	dashboarMu.Lock()
	defer dashboarMu.Unlock()

	dashboards = make(map[string]Dashboard)
}

// WriteDashboardDownloadCount makes sure data is fresh and returns the count for dashboard downloads.
func WriteDashboardDownloadCount(resp http.ResponseWriter, req *http.Request) {
	splitPaths := strings.Split(req.URL.Path, "/")
	if len(splitPaths) != dashboardPathSegments {
		http.Error(resp, "missing path segments", http.StatusNotFound)
		return
	}

	ids := strings.Split(splitPaths[4], ",")
	if len(ids) > maxDashboardIDs {
		http.Error(resp, "too many IDs", http.StatusInternalServerError)
		return
	}

	counter, fetch := checkExistingData(ids)
	if len(fetch) > 0 {
		newboards, err := fetchDashboards(req.Context(), fetch)
		if err != nil {
			http.Error(resp, "unable to get data "+err.Error(), http.StatusInternalServerError)
			return
		}

		counter += appendNewData(newboards)
	}
	// This format works with badgen.net. Subject and status are counts, not request text.
	reply := fmt.Sprintf(`{"subject": "%v dashboards", "status": %v}`, len(ids), counter)
	_, _ = resp.Write([]byte(reply)) //nolint:gosec // G705: both values are numbers.
}

// checkExistingData returns counters for fresh data, and a list of ids that need to be fetched.
func checkExistingData(ids []string) (int64, []string) {
	dashboarMu.RLock()
	defer dashboarMu.RUnlock()

	var (
		counter int64
		fetch   []string
	)

	for _, id := range ids {
		switch dashboard, ok := dashboards[id]; {
		case !ok:
			fetch = append(fetch, id)
		case time.Since(dashboard.Time) > refreshTime:
			fetch = append(fetch, id)
		default:
			counter += dashboard.Downloads
		}
	}

	return counter, fetch
}

// appendNewData locks the map and adds new or refreshed items.
func appendNewData(boards []Dashboard) int64 {
	dashboarMu.Lock()
	defer dashboarMu.Unlock()

	var counter int64

	for _, board := range boards {
		ID := strconv.FormatInt(board.ID, 10)
		dashboards[ID] = board
		counter += board.Downloads
	}

	return counter
}

// fetchDashboards returns dashboard data from the grafana api for multiple dashboards.
func fetchDashboards(ctx context.Context, ids []string) ([]Dashboard, error) {
	var (
		boards = make([]Dashboard, len(ids))
		err    error
	)

	for idx, id := range ids {
		boards[idx], err = fetchDashboard(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	return boards, nil
}

// fetchDashboards returns dashboard data from the grafana api for a single dashboard.
func fetchDashboard(ctx context.Context, dashID string) (Dashboard, error) {
	board := Dashboard{Time: time.Now()}

	_, err := strconv.ParseInt(dashID, 10, 64)
	if err != nil {
		// We only accept numbers.
		return board, fmt.Errorf("invalid dashboard ID: %s: %w", dashID, err)
	}

	url := DashboardAPI + dashID
	log.Println("Fetching", url)

	client := http.Client{Timeout: requestTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return board, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return board, fmt.Errorf("making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return board, fmt.Errorf("reading response: %w", err)
	}

	err = json.Unmarshal(body, &board)
	if err != nil {
		return board, fmt.Errorf("parsing response: %w", err)
	}

	return board, nil
}
