package main

import (
	"flag"
	"log"
	"math/rand"
	"strconv"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"

	"github.com/gin-gonic/gin"
)

func loadGraphFromData(originStationID, destStationID int) *stationgraph.StationGraph {
	origin := station.GetStationByID(originStationID)
	log.Printf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(destStationID)
	log.Printf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	graph, succeeded := stationgraph.BuildStationGraph(origin, dest)

	if !succeeded {
		log.Fatal("graph build failed")
	}

	log.Printf("found %d routes\n", graph.CountRoutes())

	log.Printf("estimating station gains...")
	skyline.InitializeEstimators(graph)

	return graph
}

func getHeuristicFromString(v string) algorithm.MonteCarloTreeRandomHeuristic {
	switch v {
	case "pbs":
		return algorithm.MonteCarloTreeRandomHeuristicPBS
	case "estimation":
		return algorithm.MonteCarloTreeRandomHeuristicEstimation
	}
	return algorithm.MonteCarloTreeRandomHeuristicUniformRandom
}

func main() {
	// log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	station.LoadStationsFromFile("./data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("./data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("./data/station_neighbors.tsv")
	skyline.LoadFlowMatrixFromFile("./data/flow_matrix.tsv")
	skyline.ComputeTimeMatrixFromDistance(station.GetDistanceMatrix())

	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	r := gin.Default()

	stationRoutes := r.Group("/api/stations")
	{
		stationRoutes.GET("/", func(context *gin.Context) {
			context.JSON(200, station.GetAllStations())
		})
	}
	routes := r.Group("/api/routes")
	{
		// The request responds to a url matching: /api/routes/?origin=1290&dest=5414&iterations=10000
		routes.GET("/", func(context *gin.Context) {
			originID, _ := strconv.Atoi(context.Query("origin"))
			destID, _ := strconv.Atoi(context.Query("dest"))

			// parameters
			P0 := 8
			Pmin := 8
			dP := 1.0
			S0 := 32768
			Smin := 256
			dS := 0.5
			c := 0.05
			N := 5
			H := getHeuristicFromString("estimation")
			A := 5.0

			graph := loadGraphFromData(originID, destID)

			start := time.Now()
			targetList := algorithm.SearchSkylineRoutesWithMCTS(graph, P0, S0, Pmin, Smin, N, dP, dS, c, H, A)
			elapsed := time.Since(start)
			log.Printf("Target | %s \t| found %d routes with %d insertions\n", elapsed, len(targetList.Routes), targetList.NumAdds)

			context.JSON(200, targetList.Routes)
		})
	}

	r.Run()
}
