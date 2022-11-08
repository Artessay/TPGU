package algorithm_test

import (
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func loadGraphFromData(t *testing.T) *stationgraph.StationGraph {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("../../data/station_neighbors.tsv")
	skyline.LoadFlowMatrixFromFile("../../data/flow_matrix.tsv")
	skyline.ComputeTimeMatrixFromDistance(station.GetDistanceMatrix())

	origin := station.GetStationByID(6842)
	log.Printf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(5664)
	log.Printf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	graph, succeeded := stationgraph.BuildStationGraph(origin, dest)

	if !succeeded {
		t.Fatal("graph build failed")
	}

	log.Printf("found %d routes\n", graph.CountRoutes())

	return graph
}

func TestMonteCarloTreeSearch(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	graph := loadGraphFromData(t)

	start := time.Now()
	skyline.InitializeEstimators(graph)
	list := algorithm.SearchSkylineRoutesWithMCTS(graph, 8, 10, 4, 256, 1, 1.0, 0.5, 1, algorithm.MonteCarloTreeRandomHeuristicEstimation, 5)
	elapsed := time.Since(start)

	list.Print()
	log.Printf("route searching costs %s\n", elapsed)

	testRoute := []*stationgraph.StationNode{graph.OriginNode}

	tree := algorithm.BuildMonteCarloTree(graph)
	tree.RandomHeuristic = algorithm.MonteCarloTreeRandomHeuristicEstimation
	weights := tree.ComputeChoiceWeights(testRoute[len(testRoute)-1].Next, testRoute)
	for i, ns := range testRoute[len(testRoute)-1].Next {
		log.Printf("Estimated weights for choice %d: %f; Hit: %d", ns.S.ID, weights[i], ns.Hit)
	}
}

func average(values []float64) float64 {
	avg := 0.0
	for _, v := range values {
		avg += v
	}
	return avg / float64(len(values))
}

func stddev(values []float64, avg float64) float64 {
	stddev := 0.0
	for _, v := range values {
		stddev += (v - avg) * (v - avg)
	}
	return math.Sqrt(stddev / float64(len(values)-1))
}

func ci(values []float64, avg float64) float64 {
	return 1.96 * stddev(values, avg) / math.Sqrt(float64(len(values)))
}

func getIntParameterFromEnv(envName string, defaultValue int) int {
	if value, exists := os.LookupEnv(envName); exists {
		if v, err := strconv.Atoi(value); err == nil {
			log.Printf("Read %s = %d from env", envName, v)
			return v
		}
	}
	log.Printf("Use default value %d for %s", defaultValue, envName)
	return defaultValue
}

func getFloatParameterFromEnv(envName string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(envName); exists {
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			log.Printf("Read %s = %.3f from env", envName, v)
			return v
		}
	}
	log.Printf("Use default value %.3f for %s", defaultValue, envName)
	return defaultValue
}

func TestCompareMCTSvsPBS(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	graph := loadGraphFromData(t)

	values := []float64{}
	bTime := []float64{}
	aTime := []float64{}

	usePBS := true
	if os.Getenv("BASELINE") == "MCSS" {
		usePBS = false
	}

	R := getIntParameterFromEnv("NUM_ROUND", 1000)
	P0 := getIntParameterFromEnv("PARAM_INIT_POOL_SIZE", 8)
	Pmin := getIntParameterFromEnv("PARAM_MIN_POOL_SIZE", 8)
	dP := getFloatParameterFromEnv("PARAM_POOL_SIZE_DECAY", 1.0)
	S0 := getIntParameterFromEnv("PARAM_INIT_NUM_ITER", 32768)
	Smin := getIntParameterFromEnv("PARAM_MIN_NUM_ITER", 256)
	dS := getFloatParameterFromEnv("PARAM_NUM_ITER_DECAY", 0.5)
	c := getFloatParameterFromEnv("PARAM_C", 0.5)
	T := getIntParameterFromEnv("PARAM_NUM_STEP", 5)

	var (
		bP0   int
		bPmin int
		bdP   float64
		bS0   int
		bSmin int
		bdS   float64
		bc    float64
		bT    int
	)

	if !usePBS {
		bP0 = getIntParameterFromEnv("BASELINE_PARAM_INIT_POOL_SIZE", 8)
		bPmin = getIntParameterFromEnv("BASELINE_PARAM_MIN_POOL_SIZE", 8)
		bdP = getFloatParameterFromEnv("BASELINE_PARAM_POOL_SIZE_DECAY", 1.0)
		bS0 = getIntParameterFromEnv("BASELINE_PARAM_INIT_NUM_ITER", 32768)
		bSmin = getIntParameterFromEnv("BASELINE_PARAM_MIN_NUM_ITER", 256)
		bdS = getFloatParameterFromEnv("BASELINE_PARAM_NUM_ITER_DECAY", 0.5)
		bc = getFloatParameterFromEnv("BASELINE_PARAM_C", 0.5)
		bT = getIntParameterFromEnv("BASELINE_PARAM_NUM_STEP", 5)
	}

	for i := 0; i < R; i++ {
		log.Printf("Round #%d...", i+1)

		start := time.Now()
		aList := algorithm.SearchSkylineRoutesWithMCTS(graph, P0, S0, Pmin, Smin, T, dP, dS, c, algorithm.MonteCarloTreeRandomHeuristicPBS, 0)
		elapsed := time.Since(start)
		log.Printf("MCTS costs %s and found %d routes with %d insertions\n", elapsed, len(aList.Routes), aList.NumAdds)
		aTime = append(aTime, float64(elapsed/time.Millisecond))

		start = time.Now()
		var bList *skyline.RouteList
		if usePBS {
			bList = algorithm.SearchSkylineRoutesWithPBS(graph, aList.NumAdds, false)
		} else {
			bList = algorithm.SearchSkylineRoutesWithMCTS(graph, bP0, bS0, bPmin, bSmin, bT, bdP, bdS, bc, algorithm.MonteCarloTreeRandomHeuristicPBS, 0)
		}
		elapsed = time.Since(start)
		log.Printf("PBS costs %s and found %d routes with %d insertions", elapsed, len(bList.Routes), bList.NumAdds)
		bTime = append(bTime, float64(elapsed/time.Millisecond))

		finalList := skyline.BuildRouteList()
		for _, r := range bList.Routes {
			finalList.Add(r.R, nil)
		}
		for _, r := range aList.Routes {
			finalList.Add(r.R, nil)
		}

		aCount := 0
		for _, r := range aList.Routes {
			if sr := finalList.GetRouteByHash(r.R, r.Hash); sr != nil {
				aCount++
			}
		}
		bCount := 0
		for _, r := range bList.Routes {
			if sr := finalList.GetRouteByHash(r.R, r.Hash); sr != nil {
				bCount++
			}
		}

		values = append(values, float64(aCount-bCount)/float64(len(finalList.Routes)))

		vavg := average(values)
		vci := ci(values, vavg)
		log.Printf("values: %v, avg = %.5f +- %.5f", values, vavg, vci)

		mtavg := average(aTime)
		mtci := ci(aTime, mtavg)
		log.Printf("mcss time: %v, avg = %.5f +- %.5f", aTime, mtavg, mtci)

		btavg := average(bTime)
		btci := ci(bTime, btavg)
		log.Printf("baseline time: %v, avg = %.5f +- %.5f", bTime, btavg, btci)
	}
}
