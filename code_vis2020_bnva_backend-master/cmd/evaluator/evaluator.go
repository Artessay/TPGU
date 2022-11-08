package main

import (
	"flag"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"

	log "github.com/sirupsen/logrus"
)

// TestResult presents test results
type TestResult struct {
	TargetTime     []float64
	BaselineTime   []float64
	TargetCount    []float64
	BaselineCount  []float64
	SkylineIndexes []float64
	Mutex          sync.Mutex
}

var stationPairPresets = [...][2]int{
	[2]int{6842, 5664},
	[2]int{1290, 1579},
	[2]int{6756, 3356},
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

func loadGraphFromData(originStationID, destStationID int, randomStations bool) *stationgraph.StationGraph {
	station.LoadStationsFromFile("./data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("./data/dist_matrix.tsv")
	station.LoadStationNeighborsFromFile("./data/station_neighbors.tsv")
	skyline.LoadFlowMatrixFromFile("./data/flow_matrix.tsv")
	skyline.ComputeTimeMatrixFromDistance(station.GetDistanceMatrix())

	if randomStations {
		log.Warning("use random origin and destination stations.")
		stations := station.GetAllStations()
		originStationID = stations[rand.Intn(len(stations))].ID
		destStationID = stations[rand.Intn(len(stations))].ID
	}

	origin := station.GetStationByID(originStationID)
	log.WithFields(log.Fields{
		"id":  origin.ID,
		"lon": origin.Lon,
		"lat": origin.Lat,
	}).Info("origin station selected")

	dest := station.GetStationByID(destStationID)
	log.WithFields(log.Fields{
		"id":  dest.ID,
		"lon": dest.Lon,
		"lat": dest.Lat,
	}).Info("destination station selected")

	graph, succeeded := stationgraph.BuildStationGraph(origin, dest)

	if !succeeded {
		log.Panic("graph build failed")
	}

	log.WithField("num", graph.CountRoutes()).Info("the number of routes counted")

	graph.CountChoices()
	log.Info("done")

	log.Info("estimating station gains...")
	start := time.Now()
	skyline.InitializeEstimators(graph)
	targetElapsed := time.Since(start)
	log.Info("estimators ready.")
	log.WithFields(log.Fields{
		"time": targetElapsed,
	}).Info("target completed")

	return graph
}

type heterogeneousSkylineRoute struct {
	Origin int
	Route  *skyline.Route
}

// the pareto-set difference, larger shows target is better
func calculateDifference(target, baseline *skyline.RouteList) float64 {
	switch {
	case len(target.Routes) == 0 && len(baseline.Routes) == 0:
		return 0
	case len(target.Routes) == 0:
		return -1
	case len(baseline.Routes) == 0:
		return 1
	}

	var hsr []*heterogeneousSkylineRoute
	for _, r := range target.Routes {
		hsr = append(hsr, &heterogeneousSkylineRoute{Origin: 1, Route: r})
	}
	for _, r := range baseline.Routes {
		hsr = append(hsr, &heterogeneousSkylineRoute{Origin: 2, Route: r})
	}

	sort.Slice(hsr, func(i, j int) bool {
		if math.Abs(hsr[i].Route.Criteria[1]-hsr[j].Route.Criteria[1]) < 1e-7 {
			return hsr[i].Route.Criteria[0] < hsr[j].Route.Criteria[0]
		}
		return hsr[i].Route.Criteria[1] > hsr[j].Route.Criteria[1]
	})

	bestTime := hsr[0].Route.Criteria[0]
	bestFlow := hsr[0].Route.Criteria[1]

	numAccepted := make([]int, 3)
	numAccepted[0] = 1
	numAccepted[hsr[0].Origin] = 1
	for _, r := range hsr[1:] {
		// log.Infof("BT %f T %f BF %f F %f", bestTime, r.Route.Criteria[0], bestFlow, r.Route.Criteria[1])
		if math.Abs(r.Route.Criteria[1]-bestFlow) < 1e-7 && math.Abs(r.Route.Criteria[0]-bestTime) < 1e-7 {
			numAccepted[r.Origin]++
			// log.Infof("same %d", r.Origin)
		} else if r.Route.Criteria[0] < bestTime-1e-7 {
			numAccepted[r.Origin]++
			numAccepted[0]++
			bestTime = r.Route.Criteria[0]
			bestFlow = r.Route.Criteria[1]
			// log.Infof("add %d", r.Origin)
		}
	}

	// log.Infof("accepted %v", numAccepted)
	return float64(numAccepted[1]-numAccepted[2]) / float64(numAccepted[0])
}

func runTest(
	k int,
	graph *stationgraph.StationGraph,
	baseline string,
	P0, Pmin int,
	dP float64,
	S0, Smin int,
	dS float64,
	c float64,
	N int,
	H algorithm.MonteCarloTreeRandomHeuristic,
	A float64,
	bP0, bPmin int,
	bdP float64,
	bS0, bSmin int,
	bdS float64,
	bc float64,
	bN int,
	bH algorithm.MonteCarloTreeRandomHeuristic,
	bA float64,
	result *TestResult,
	printSkyline bool,
) {
	start := time.Now()
	targetList := algorithm.SearchSkylineRoutesWithMCTS(graph, P0, S0, Pmin, Smin, N, dP, dS, c, H, A)
	targetElapsed := time.Since(start)

	log.WithFields(log.Fields{
		"round":      k,
		"time":       targetElapsed,
		"insertions": targetList.NumAdds,
		"deletion":   targetList.NumDeletion,
		"count":      len(targetList.Routes),
	}).Info("target completed")

	if printSkyline {
		targetList.Print()
	}

	start = time.Now()
	var baselineList *skyline.RouteList
	switch baseline {
	case "pbs":
		baselineList = algorithm.SearchSkylineRoutesWithPBS(graph, targetList.NumAdds, false)
	case "mcss":
		baselineList = algorithm.SearchSkylineRoutesWithMCTS(graph, bP0, bS0, bPmin, bSmin, bN, bdP, bdS, bc, bH, bA)
	}
	baselineElapsed := time.Since(start)
	log.WithFields(log.Fields{
		"round":      k,
		"time":       baselineElapsed,
		"insertions": baselineList.NumAdds,
		"count":      len(baselineList.Routes),
	}).Info("baseline completed")

	if printSkyline {
		baselineList.Print()
	}

	diff := calculateDifference(targetList, baselineList)

	result.Mutex.Lock()
	result.TargetTime = append(result.TargetTime, float64(targetElapsed/time.Millisecond))
	result.TargetCount = append(result.TargetCount, float64(len(targetList.Routes)))
	result.BaselineTime = append(result.BaselineTime, float64(baselineElapsed/time.Millisecond))
	result.BaselineCount = append(result.BaselineCount, float64(len(baselineList.Routes)))
	result.SkylineIndexes = append(result.SkylineIndexes, diff)
	result.Mutex.Unlock()
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
	numRound := flag.Int("round", 1, "the number of testing round")
	jobs := flag.Int("jobs", 4, "parallelize tests")
	randomStations := flag.Bool("rand", false, "use random origin and destination stations")
	stationPreset := flag.Int("preset", 0, "station preset")
	printSkyline := flag.Bool("print", false, "print skyline set")

	P0 := flag.Int("p0", 8, "initial pool size")
	Pmin := flag.Int("pmin", 8, "minimum pool size")
	dP := flag.Float64("dp", 1.0, "pool size decay rate")
	S0 := flag.Int("s0", 32768, "initial number of iterations")
	Smin := flag.Int("smin", 256, "minimum number of iterations")
	dS := flag.Float64("ds", 0.7, "number of iteration decay rate")
	c := flag.Float64("c", 0.005, "c value, larger would turn to exhausted search")
	N := flag.Int("n", 10, "number of descending steps")
	H := flag.String("h", "estimation", "random heuristic")
	A := flag.Float64("a", 5, "bias in the estimation heuristic")

	baseline := flag.String("baseline", "pbs", "the baseline method to compare with (pbs, mcss)")
	bP0 := flag.Int("bp0", 8, "baseline (mcss) initial pool size")
	bPmin := flag.Int("bpmin", 8, "baseline (mcss) minimum pool size")
	bdP := flag.Float64("bdp", 1.0, "baseline (mcss) pool size decay rate")
	bS0 := flag.Int("bs0", 16384, "baseline (mcss) initial number of iterations")
	bSmin := flag.Int("bsmin", 256, "baseline (mcss) minimum number of iterations")
	bdS := flag.Float64("bds", 0.5, "baseline (mcss) number of iteration decay rate")
	bc := flag.Float64("bc", 0.01, "baseline (mcss) c value")
	bN := flag.Int("bn", 10, "baseline (mcss) number of descending steps")
	bH := flag.String("bh", "estimation", "baseline (mcss) random heuristic")
	bA := flag.Float64("ba", 5, "baseline (mcss) bias in the estimation heuristic")

	flag.Parse()

	// Performance Profiling
	f, err := os.Create("cpuprofile")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	log.WithFields(log.Fields{
		"round":  *numRound,
		"jobs":   *jobs,
		"rand":   *randomStations,
		"preset": *stationPreset,
		"print":  *printSkyline,
	}).Info("the evaluation has started.")

	log.WithFields(log.Fields{
		"p0":   *P0,
		"pmin": *Pmin,
		"dp":   *dP,
		"s0":   *S0,
		"smin": *Smin,
		"dS":   *dS,
		"c":    *c,
		"n":    *N,
		"h":    *H,
		"a":    *A,
	}).Info("target algorithm ready")

	if *baseline == "pbs" {
		log.Info("baseline algorithm (pbs) ready")
	} else {
		log.WithFields(log.Fields{
			"p0":   *bP0,
			"pmin": *bPmin,
			"dp":   *bdP,
			"s0":   *bS0,
			"smin": *bSmin,
			"dS":   *bdS,
			"c":    *bc,
			"n":    *bN,
			"h":    *bH,
			"a":    *bA,
		}).Info("baseline algorithm (mcts) ready")
	}

	if *stationPreset >= len(stationPairPresets) || *stationPreset < 0 {
		log.Panic("bad station preset specified")
	}

	rand.Seed(time.Now().UnixNano())

	graph := loadGraphFromData(stationPairPresets[*stationPreset][0], stationPairPresets[*stationPreset][1], *randomStations)

	// add stop randomly
	// rand.Seed(time.Now().UnixNano())
	// stop := rand.Intn(len(graph.Nodes))
	// for {
	// 	if graph.Nodes[stop].S.ID != graph.OriginNode.S.ID && graph.Nodes[stop].S.ID != graph.DestNode.S.ID {
	// 		break
	// 	}
	// 	stop = rand.Intn(len(graph.Nodes))
	// }
	// graph.AddStop(graph.Nodes[stop])
	// log.Printf("use %d (%f, %f) as a stop", graph.Nodes[stop].S.ID, graph.Nodes[stop].S.Lon, graph.Nodes[stop].S.Lat)
	// graph = stationgraph.BuildSubGraph(graph)
	// log.Printf("the number of routes after setting stop: %v \n", graph.CountRoutes())

	var wg sync.WaitGroup
	limit := make(chan struct{}, *jobs)
	result := TestResult{}

	// test with different iteration number
	wg.Add(*numRound)
	for i := 0; i < *numRound; i++ {
		k := i + 1
		limit <- struct{}{}
		go func() {
			runTest(k, graph, *baseline,
				*P0, *Pmin, *dP, *S0, *Smin, *dS, *c, *N, getHeuristicFromString(*H), *A,
				*bP0, *bPmin, *bdP, *bS0, *bSmin, *bdS, *bc, *bN, getHeuristicFromString(*bH), *bA,
				&result, *printSkyline)
			<-limit
			wg.Done()
		}()
	}
	wg.Wait()

	indexAverage := average(result.SkylineIndexes)
	indexCI := ci(result.SkylineIndexes, indexAverage)
	log.WithFields(log.Fields{
		"average": indexAverage,
		"ci":      indexCI,
		"samples": result.SkylineIndexes,
	}).Info("skyline indexes")

	targetCountAverage := average(result.TargetCount)
	targetCountCI := ci(result.TargetCount, targetCountAverage)
	log.WithFields(log.Fields{
		"average": targetCountAverage,
		"ci":      targetCountCI,
		"samples": result.TargetCount,
	}).Info("target count")

	targetTimeAverage := average(result.TargetTime)
	targetTimeCI := ci(result.TargetTime, targetTimeAverage)
	log.WithFields(log.Fields{
		"average": targetTimeAverage,
		"ci":      targetTimeCI,
		"samples": result.TargetTime,
	}).Info("target time")

	baselineCountAverage := average(result.BaselineCount)
	baselineCountCI := ci(result.BaselineCount, baselineCountAverage)
	log.WithFields(log.Fields{
		"average": baselineCountAverage,
		"ci":      baselineCountCI,
		"samples": result.BaselineCount,
	}).Info("baseline count")

	baselineTimeAverage := average(result.BaselineTime)
	baselineTimeCI := ci(result.BaselineTime, baselineTimeAverage)
	log.WithFields(log.Fields{
		"average": baselineTimeAverage,
		"ci":      baselineTimeCI,
		"samples": result.BaselineTime,
	}).Info("baseline time")
}
