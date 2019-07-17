package grafana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DashboardAPI is the URL to the JSON API at Grafana.com
const DashboardAPI = "https://grafana.com/api/dashboards/%v"

const refreshTime = time.Hour

// Dashboard holds a dashboard's name and download count.
// This is a small snippet of the data available from the Grafana API.
type Dashboard struct {
	Name      string    `json:"name"`
	ID        int64     `json:"id"`
	Downloads int64     `json:"downloads"`
	Ts        time.Time `json:"-"`
}

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
func WriteDashboardDownloadCount(w http.ResponseWriter, r *http.Request) {
	splitPaths := strings.Split(r.URL.Path, "/")
	if len(splitPaths) != 5 {
		http.Error(w, "missing path segments", http.StatusNotFound)
		return
	}
	ids := strings.Split(splitPaths[4], ",")
	if len(ids) > 50 {
		http.Error(w, "too many IDs", http.StatusInternalServerError)
		return
	}

	counter, fetch := checkExistingData(ids)
	if len(fetch) > 0 {
		newboards, err := fetchDashboards(fetch)
		if err != nil {
			http.Error(w, "unable to get data "+err.Error(), http.StatusInternalServerError)
			return
		}
		counter += appendNewData(newboards)
	}
	// This format works with badgen.net.
	reply := fmt.Sprintf(`{"subject": "%v dashboards", "status": %v}`, len(ids), counter)
	_, _ = w.Write([]byte(reply))
}

// checkExistingData returns counters for fresh data, and a list of ids that need to be fetched.
func checkExistingData(ids []string) (counter int64, fetch []string) {
	dashboarMu.RLock()
	defer dashboarMu.RUnlock()

	for _, id := range ids {
		switch dashboard, ok := dashboards[id]; {
		case !ok:
			fetch = append(fetch, id)
		case time.Since(dashboard.Ts) > refreshTime:
			fetch = append(fetch, id)
		default:
			counter += dashboard.Downloads
		}
	}
	return
}

// appendNewData locks the map and adds new or refreshed items.
func appendNewData(boards []Dashboard) (counter int64) {
	dashboarMu.Lock()
	defer dashboarMu.Unlock()
	for _, board := range boards {
		ID := strconv.FormatInt(board.ID, 10)
		dashboards[ID] = board
		counter += board.Downloads
	}
	return
}

// fetchDashboards returns dashboard data from the grafana api for multiple dashboards.
func fetchDashboards(ids []string) ([]Dashboard, error) {
	boards := make([]Dashboard, len(ids))
	var err error
	for i, id := range ids {
		if boards[i], err = fetchDashboard(id); err != nil {
			return nil, err
		}
	}
	return boards, nil
}

// fetchDashboards returns dashboard data from the grafana api for a single dashboard.
func fetchDashboard(id string) (Dashboard, error) {
	board := Dashboard{Ts: time.Now()}
	URL := fmt.Sprintf(DashboardAPI, id)
	log.Println("Fetching", URL)

	resp, err := http.Get(URL)
	if err != nil {
		return board, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return board, err
	}
	return board, json.Unmarshal(body, &board)
}
