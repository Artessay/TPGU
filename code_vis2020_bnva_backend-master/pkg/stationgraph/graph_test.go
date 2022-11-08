package stationgraph_test

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func TestBuildStationGraph(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("../../data/station_neighbors.tsv")

	origin := station.GetStationByID(9392)
	t.Logf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(10268)
	t.Logf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	graph, succeeded := stationgraph.BuildStationGraph(origin, dest)

	if !succeeded {
		t.Fatal("graph build failed")
	}

	log.Printf("total %v nodes in the graph\n", len(graph.Nodes))
	log.Printf("found %v route to the destination\n", graph.CountRoutes())

	rand.Seed(time.Now().UnixNano())
	stop := rand.Intn(len(graph.Nodes))
	// stop := 8898
	for {
		if graph.Nodes[stop].S.ID != origin.ID && graph.Nodes[stop].S.ID != dest.ID {
			break
		}
		stop = rand.Intn(len(graph.Nodes))
	}
	graph.AddStop(graph.Nodes[stop])
	log.Printf("use %d (%f, %f) as a stop", graph.Nodes[stop].S.ID, graph.Nodes[stop].S.Lon, graph.Nodes[stop].S.Lat)

	ng := stationgraph.BuildSubGraph(graph)
	log.Printf("total %v nodes in the sub graph\n", len(ng.Nodes))
	log.Printf("found %v route to the destination in the sub graph\n", ng.CountRoutes())

	// combine subgraphs
	bigGraph, succeeded := stationgraph.BuildGraphWithStops(origin, dest, []*station.Station{station.GetStationByID(8898)})
	if !succeeded {
		t.Fatal("graph build failed")
	}

	log.Printf("total %v nodes in the combined graph\n", len(bigGraph.Nodes))
	log.Printf("found %v route to the destination in the combined graph\n", bigGraph.CountRoutes())

}
