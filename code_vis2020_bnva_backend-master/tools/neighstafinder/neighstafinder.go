package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

type queryResult struct {
	ID        int
	Neighbors []int
}

const deltaDist = 3000.0
const maxNumStations = 2000

func query(db *sql.DB, id int, lon float64, lat float64) queryResult {
	rows, err := db.Query(
		`SELECT id FROM bus_stations_unique_slim WHERE location <-> ST_GEOGFROMTEXT('POINT(' || $1 || ' ' || $2 || ')') < $3 and id != $4`,
		lon, lat, deltaDist, id)
	if err != nil {
		log.Fatalln("query failed:", err)
	}

	var result queryResult
	result.ID = id

	for rows.Next() {
		var nid int
		rows.Scan(&nid)
		result.Neighbors = append(result.Neighbors, nid)
	}

	log.Printf("[%d] found %d neighbors.\n", id, len(result.Neighbors))
	return result
}

func main() {
	log.Println("connecting to postgresql...")
	connStr := "postgres://rnvis:pPvqMgnBoaDvn4nxdADYiWwr@postgres.zjuvis.org/urbanvis_bus_trips_beijing_2013?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalln("unable to connect to postgresql!", err)
	}

	stationFilePath := "data/bus_stations.tsv"
	log.Println("loading stations from", stationFilePath)

	fp, err := os.Open(stationFilePath)
	if err != nil {
		log.Fatal("cannot open file", stationFilePath, err)
	}
	defer fp.Close()

	var wg sync.WaitGroup
	guard := make(chan struct{}, 5)
	gather := make(chan queryResult, maxNumStations)
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		id, _ := strconv.Atoi(columns[0])
		lon, _ := strconv.ParseFloat(columns[2], 64)
		lat, _ := strconv.ParseFloat(columns[3], 64)

		wg.Add(1)
		guard <- struct{}{}
		go func() {
			gather <- query(db, id, lon, lat)
			wg.Done()
			<-guard
		}()
	}

	fout, err := os.Create("data/station_neighbors.tsv")
	if err != nil {
		log.Fatalf("cannot open file to write!\n")
	}
	defer fout.Close()

	wg.Wait()
	stop := false
	for !stop {
		select {
		case result, _ := <-gather:
			var nb string
			if len(result.Neighbors) > 0 {
				nb = strings.Trim(strings.Replace(fmt.Sprint(result.Neighbors), " ", ",", -1), "[]")
			} else {
				nb = "nil"
			}
			fmt.Fprintf(fout, "%d\t%s\n", result.ID, nb)
		default:
			stop = true
		}
	}
}
