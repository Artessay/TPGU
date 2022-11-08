package main

// input format (csv, wgs84):
// id, lat, lng

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	transform "github.com/googollee/eviltransform/go"
	log "github.com/sirupsen/logrus"
)

// Location represents a location
type Location struct {
	ID  int64
	Lon float64
	Lat float64
}

const (
	QueryStatusOK      = 0
	QueryStatusSkipped = 1
	QueryStatusError   = 2
)

// Query represents distance query
type Query struct {
	ReqSeq   int
	TotalReq int
	Src      *Location
	Dst      *Location
	Distance float64
	Status   int
}

// QueryDistance queries the distance from OSRM
func QueryDistance(q *Query) {
	logger := log.New().WithField("id", q.ReqSeq).WithField("total", q.TotalReq)
	if dist := transform.Distance(q.Src.Lat, q.Src.Lon, q.Dst.Lat, q.Dst.Lon); dist < 100 || dist > 5000 {
		q.Status = QueryStatusSkipped
		logger.Debugf("skipped because the distance (%f meters) is not within the threshold\n", dist)
		return
	}

	for retries := 0; retries < 10; retries++ {
		uri := fmt.Sprintf(
			"http://172.24.37.185:5000/route/v1/driving/%f,%f;%f,%f",
			q.Src.Lon, q.Src.Lat, q.Dst.Lon, q.Dst.Lat)

		logger.Debugf("querying %s...\n", uri)

		resp, err := http.Get(uri)

		if err != nil {
			logger.Errorf("request failed: %s\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		if result["code"] == nil {
			logger.Errorf("no 'code' field found!\n")
			time.Sleep(5 * time.Second)
			continue
		}

		switch result["code"] {
		case "Ok":
			q.Distance = result["routes"].([]interface{})[0].(map[string]interface{})["distance"].(float64)
			logger.Debugf("The distance is %.3f meters", q.Distance)
			return
		case "NoRoute":
			logger.Errorf("[%d,%d] Error: no route found!\n", q.ReqSeq, q.TotalReq)
			q.Status = QueryStatusSkipped
			return
		default:
			logger.Errorf("server returned %s!\n", result["code"])
			time.Sleep(5 * time.Second)
		}
	}

	q.Status = QueryStatusError
}

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Printf("Usage: %s location_file output_file", os.Args[0])
		return
	}

	fin, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("cannot open location file!")
	}
	defer fin.Close()

	fout, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal("cannot open output file!\n")
	}
	defer fout.Close()

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	var locations []*Location
	r := csv.NewReader(fin)
	var header []string
	var headerMapping map[string]int
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}

		if header == nil {
			header = record
			headerMapping = make(map[string]int)
			for i, name := range header {
				headerMapping[name] = i
			}
		} else {
			id, _ := strconv.ParseInt(record[headerMapping["id"]], 10, 64)
			lat, _ := strconv.ParseFloat(record[headerMapping["wgs_lat"]], 64)
			lon, _ := strconv.ParseFloat(record[headerMapping["wgs_lon"]], 64)

			locations = append(locations, &Location{ID: id, Lat: lat, Lon: lon})
		}
	}

	var wg sync.WaitGroup
	var results []*Query

	guard := make(chan struct{}, 20)

	if len(os.Args) == 4 {
		parts := strings.Split(os.Args[3], ";")
		var pairs [][]int
		for _, p := range parts {
			srcdst := strings.Split(p, ",")
			src, _ := strconv.Atoi(srcdst[0])
			dst, _ := strconv.Atoi(srcdst[1])
			pairs = append(pairs, []int{src, dst})
		}

		totalRequestCount := len(pairs)
		requestSequence := 0
		for _, p := range pairs {
			requestSequence++

			q := &Query{
				ReqSeq:   requestSequence,
				TotalReq: totalRequestCount,
				Src:      locations[p[0]],
				Dst:      locations[p[1]],
			}

			wg.Add(1)
			results = append(results, q)

			guard <- struct{}{}
			go func() {
				QueryDistance(q)
				wg.Done()
				<-guard
			}()
		}
	} else {
		totalRequestCount := len(locations) * len(locations)
		requestSequence := 0
		for i := range locations {
			for j := range locations {
				requestSequence++

				q := &Query{
					ReqSeq:   requestSequence,
					TotalReq: totalRequestCount,
					Src:      locations[i],
					Dst:      locations[j],
				}

				wg.Add(1)
				results = append(results, q)

				guard <- struct{}{}
				go func() {
					QueryDistance(q)
					wg.Done()
					<-guard
				}()
			}
		}
	}

	wg.Wait()

	var failedIDPairs []string
	var skippedCounter int
	fmt.Fprintf(fout, "src_id,dst_id,distance_in_meters")
	for _, result := range results {
		switch result.Status {
		case QueryStatusOK:
			fmt.Fprintf(fout, "%d,%d,%f\n", result.Src.ID, result.Dst.ID, result.Distance)
		case QueryStatusError:
			failedIDPairs = append(failedIDPairs, fmt.Sprintf("%d,%d", result.Src.ID, result.Dst.ID))
		case QueryStatusSkipped:
			skippedCounter++
		}
	}
	if skippedCounter > 0 {
		fmt.Printf("%d queries skipped.", skippedCounter)
	}
	if len(failedIDPairs) != 0 {
		fmt.Printf("%d queries failed. These queries may be restarted with: %s", len(failedIDPairs), strings.Join(failedIDPairs, ";"))
	}
}
