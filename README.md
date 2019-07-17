# badgedata

Simple Go library to collect remote data for use as badgen badge source data.

```shell
go get code.golift.io/badgedata
```

## Example

Simple example to show how to use it. You should put this library into your own
web server code and give it a handler path you prefer. Has a simple pluggable
structure to make creating new data sources simple. Contains one example for
caching grafana dashboard download counts.

```go
package main

import (
	"net/http"

	"code.golift.io/badgedata"
	_ "code.golift.io/badgedata/grafana"
)

func main() {
	http.Handle("/bd/", badgedata.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
```

Currently only does one thing.
```shell
curl http://127.0.0.1:8080/bd/grafana/dashboard-count/10418,10417,10416,10415
```

Replace the numbers with IDs of dashboards on Grafana.com you want download counts for.
