package algorithm_test

import (
	"log"
	"testing"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func TestSearchSkylineRoutesBruteForce(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("../../data/station_neighbors.tsv")
	skyline.LoadFlowMatrixFromFile("../../data/flow_matrix.tsv")
	skyline.ComputeTimeMatrixFromDistance(station.GetDistanceMatrix())

	origin := station.GetStationByID(6842)
	t.Logf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(5664)
	t.Logf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	graph, succeeded := stationgraph.BuildStationGraph(origin, dest)

	if !succeeded {
		t.Fatal("graph build failed")
	}

	log.Printf("found %d routes\n", graph.CountRoutes())

	start := time.Now()
	routeList := algorithm.SearchSkylineRoutesBruteForce(graph)
	elapsed := time.Since(start)

	routeList.Print()
	log.Printf("route searching costs %s\n", elapsed)
}
