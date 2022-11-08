package gqlapi

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "bustrips"
)

// ConnectToPostgre connect to the database
func ConnectToPostgre() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	log.Printf("successfully connect to Postgresql database, %v", psqlInfo)
	return db
}

// GetAllExisitedBusRoutes get all bus routes
func GetAllExisitedBusRoutes(db *sql.DB) []*ExistBusRoute {
	rows, err := db.Query("select * from bus_routes where array_length(stations, 1) <> 0")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	busRoutes := []*ExistBusRoute{}
	for rows.Next() {
		route := ExistBusRoute{}
		stations := []int64{}

		err := rows.Scan(&route.Name, (*pq.Int64Array)(&stations))

		if err != nil {
			log.Printf("%v", err)
			log.Printf("Stations % v: %v", route.Name, stations)
			continue
		}
		for _, s := range stations {
			route.Stations = append(route.Stations, int(s))
		}

		busRoutes = append(busRoutes, &route)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return busRoutes
}
