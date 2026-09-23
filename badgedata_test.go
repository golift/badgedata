package badgedata_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golift.io/badgedata"
)

func TestHandlerRoutes(t *testing.T) {
	t.Parallel()

	badgedata.Register("ping", func(resp http.ResponseWriter, _ *http.Request) {
		_, _ = resp.Write([]byte("pong"))
	})

	handler := badgedata.Handler()

	// Registrations after Handler is built are not visible to that handler.
	badgedata.Register("late", func(resp http.ResponseWriter, _ *http.Request) {
		_, _ = resp.Write([]byte("too late"))
	})

	tests := []struct {
		path   string
		status int
		body   string
	}{
		{path: "/badgedata", status: http.StatusNotFound, body: "missing path segments\n"},
		{path: "/badgedata/missing", status: http.StatusNotFound, body: "not found: missing\n"},
		{path: "/badgedata/ping", status: http.StatusOK, body: "pong"},
		{path: "/badgedata/late", status: http.StatusNotFound, body: "not found: late\n"},
	}

	for _, item := range tests {
		item := item

		t.Run(item.path, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, item.path, http.NoBody)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != item.status {
				t.Fatalf("status %d, want %d, body %q", rec.Code, item.status, rec.Body.String())
			}

			if rec.Body.String() != item.body {
				t.Fatalf("body %q, want %q", rec.Body.String(), item.body)
			}
		})
	}
}
