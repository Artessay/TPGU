package main

import (
	"log"
	"net/http"
	"os"

	"git.zjuvis.org/rnvis/bus-routing-backend/cmd/gqlapi"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/handlers"

	"flag"
	"math/rand"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"github.com/gorilla/websocket"
)

const defaultPort = "8089"

func main() {
	gqlapi.LoadLocationsFromCSVFile("./data/bus_stations_wgs84.csv")
	station.LoadStationsFromFile("./data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("./data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("./data/station_neighbors.tsv")
	skyline.LoadFlowMatrixFromFile("./data/flow_matrix.tsv")
	skyline.ComputeTimeMatrixFromDistance(station.GetDistanceMatrix())

	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	// websocket
	var upgrader = websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	options := []handler.Option{
		handler.WebsocketUpgrader(upgrader),
	}

	handler.WebsocketKeepAliveDuration(10 * time.Second)

	// graphql api
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	gqlHandler := handler.GraphQL(
		gqlapi.NewExecutableSchema(gqlapi.New()),
		options...,
	)

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))

	// provide two equal endpoint
	http.Handle("/query", gqlHandler)
	http.Handle("/graphql", gqlHandler)

	// http.Handle("/query", handler.GraphQL(
	// 	gqlapi.NewExecutableSchema(gqlapi.New()),
	// 	options...,
	// ))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))
}
