package skyline_test

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func TestEstimators(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

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

	skyline.InitializeEstimators(graph)
	testRoute := []*stationgraph.StationNode{graph.OriginNode}

	tree := algorithm.BuildMonteCarloTree(graph)
	tree.RandomHeuristic = algorithm.MonteCarloTreeRandomHeuristicEstimation
	weights := tree.ComputeChoiceWeights(testRoute[len(testRoute)-1].Next, testRoute)
	for i, ns := range testRoute[len(testRoute)-1].Next {
		log.Printf("Estimated weights for choice %d: %f", ns.S.ID, weights[i])
	}

	// type Estimation struct {
	// 	id       int
	// 	criteria []float64
	// }

	// var estimations []Estimation
	// for _, ns := range testRoute[len(testRoute)-1].Next {
	// 	log.Printf("Estimating gains for choice %d:", ns.S.ID)
	// 	values := make([]float64, len(algorithm.SkylineRouteCriteria))
	// 	for i, c := range algorithm.SkylineRouteCriteria {
	// 		values[i] = c.PredictStationGain(testRoute, ns)
	// 	}
	// 	estimations = append(estimations, Estimation{id: ns.S.ID, criteria: values})
	// }

	// sort.Slice(estimations, func(i, j int) bool { return estimations[i].criteria[1] > estimations[j].criteria[1] })
	// for _, est := range estimations {
	// 	log.Printf("Station %d: %v", est.id, est.criteria)
	// }

	// algorithm.BuildSkylineRouteList()
	// sorted := graph.SortTopo()
	// for _, s := range sorted {
	// 	log.Printf("ID: %d, EstimatedGain: %v", s.S.ID, s.EstimatedGain)
	// }
}
