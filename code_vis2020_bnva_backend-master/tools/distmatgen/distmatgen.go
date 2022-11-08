package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	transform "github.com/googollee/eviltransform/go"
)

// Query distance query
type Query struct {
	SrcID    int
	SrcLon   float64
	SrcLat   float64
	DstID    int
	DstLon   float64
	DstLat   float64
	Distance float64
}

// QueryDistance queries the distance from OSRM
func QueryDistance(q *Query) {
	gslat, gslon := transform.GCJtoWGSExact(q.SrcLat, q.SrcLon)
	gdlat, gdlon := transform.GCJtoWGSExact(q.DstLat, q.DstLon)

	for retries := 0; retries < 5; retries++ {
		log.Printf(
			"[%06d,%06d,%d/5] Querying distance between (%.3f, %.3f) and (%.3f, %.3f)...\n",
			q.SrcID, q.DstID, retries+1, gslat, gslon, gdlat, gdlon)

		uri := fmt.Sprintf(
			"http://localhost:5000/route/v1/driving/%f,%f;%f,%f",
			gslon, gslat, gdlon, gdlat)
		resp, err := http.Get(uri)

		if err != nil {
			log.Printf("[%06d,%06d,%d/5] ERROR query failed.\n", q.SrcID, q.DstID, retries+1)
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		if result["code"] == nil {
			log.Printf("[%06d,%06d,%d/5] ERROR no 'code' field in results.\n", q.SrcID, q.DstID, retries+1)
			time.Sleep(5 * time.Second)
			continue
		}

		if result["code"] != "Ok" {
			log.Printf("[%06d,%06d,%d/5] ERROR server returned %v.\n", q.SrcID, q.DstID, retries+1, result["code"])
			time.Sleep(5 * time.Second)
			continue
		}

		if len(result["routes"].([]interface{})) == 0 {
			log.Printf("[%06d,%06d,%d/5] WARNING no route returned.", q.SrcID, q.DstID, retries+1)
			q.Distance = 0
		} else {
			q.Distance = result["routes"].([]interface{})[0].(map[string]interface{})["distance"].(float64)
		}

		log.Printf("[%06d,%06d,%d/5] The distance between (%.3f, %.3f) and (%.3f, %.3f) is %.3f",
			q.SrcID, q.DstID, retries+1, gslat, gslon, gdlat, gdlon, q.Distance)
		return
	}
}

func main() {
	fp, err := os.Open("../data/station_position.tsv")
	if err != nil {
		log.Fatalln("cannot open position file!")
	}
	defer fp.Close()

	var wg sync.WaitGroup
	var results []*Query

	guard := make(chan struct{}, 10)

	for {
		var q Query
		_, err := fmt.Fscan(fp, &q.SrcID, &q.SrcLon, &q.SrcLat, &q.DstID, &q.DstLon, &q.DstLat)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("fail to read position file!\n")
		}

		wg.Add(1)
		results = append(results, &q)

		guard <- struct{}{}
		go func() {
			QueryDistance(&q)
			wg.Done()
			<-guard
		}()
	}

	wg.Wait()

	fout, err := os.Create("../data/dist_matrix.tsv")
	if err != nil {
		log.Fatalf("cannot open file to write!\n")
	}
	defer fout.Close()

	for i := range results {
		fmt.Fprintf(fout, "%d\t%d\t%f\n", results[i].SrcID, results[i].DstID, results[i].Distance)
	}
}
