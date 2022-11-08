package gqlapi

import (
	"context"
	"encoding/csv"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	geo "github.com/paulmach/go.geo"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"

	osrm "github.com/gojuno/go.osrm"
)

var stationGraph stationgraph.StationGraph

// Location lat and lon
type Location struct {
	ID  int
	Lat float64
	Lon float64
}

// LocationIndex location id index
var LocationIndex map[int]*Location

var mctsParam = struct {
	maxPoolSize, maxNumIter int
	minPoolSize, minNumIter int
	numSteps                int
	poolSizeDecay           float64
	numIterDecay            float64
	c                       float64
	heuristic               algorithm.MonteCarloTreeRandomHeuristic
	alpha                   float64
}{
	maxPoolSize:   8,
	minPoolSize:   8,
	poolSizeDecay: 1.0,
	// maxNumIter:    32768,
	maxNumIter:   8192,
	minNumIter:   256,
	numIterDecay: 0.5,
	c:            0.05,
	numSteps:     5,
	heuristic:    getHeuristicFromString("estimation"),
	alpha:        5.0}

const sampleInterval = 512

// SearchSkylineRoutesWithMCTS run MCTS on the server side with available subscription
func SearchSkylineRoutesWithMCTS(
	graph *stationgraph.StationGraph,
	tree *algorithm.MonteCarloTree,
	channel chan *algorithm.MonteCarloTree,
	list *skyline.RouteList,
) {
	tree.RandomHeuristic = mctsParam.heuristic
	tree.Alpha = mctsParam.alpha
	tree.Rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	poolsize := float64(mctsParam.maxPoolSize) + 1e-7
	numiter := float64(mctsParam.maxNumIter) + 1e-7
	pool := []*algorithm.MonteCarloTreeNode{tree.Root}
	for k := 0; k < mctsParam.numSteps; k++ {
		for _, n := range pool {
			maxi := int(numiter)
			for i := 0; i < maxi; i++ {
				if succeeded := tree.ExploreOnce(n, list, mctsParam.c); !succeeded {
					break
				}

				// Update the MCT through channel
				// log.Printf("explore %d times...", i)
				if i%(sampleInterval*(k+1)) == 0 {
					// compute uct for nodes
					for _, node := range tree.Nodes {
						node.Values.UValue = tree.ComputeUCT(node, mctsParam.c)
					}

					channel <- tree
					log.Printf("Put data in the channel...")
					// tree.Print(tree.Root, )
				}
			}
		}

		if k != mctsParam.numSteps-1 {
			var newpool []*algorithm.MonteCarloTreeNode
			for _, n := range pool {
				for _, a := range n.Actions {
					if a.Values.W > 0 {
						newpool = append(newpool, a)
					}
				}
			}

			sort.Slice(newpool, func(i, j int) bool { return newpool[i].Values.W > newpool[j].Values.W })
			if len(newpool) > int(poolsize) {
				pool = newpool[:int(poolsize)]
			} else {
				pool = newpool
			}

			// Decay for smaller route subspaces
			if int(poolsize*mctsParam.poolSizeDecay) >= mctsParam.minPoolSize {
				poolsize *= mctsParam.poolSizeDecay
			}
			if int(numiter*mctsParam.numIterDecay) >= mctsParam.minNumIter {
				numiter *= mctsParam.numIterDecay
			}
		}
	}

	tree.Done = true
	log.Printf("Find Route Lists")
}

// LoadGraphFromData compute a station graph from given o, d and stops
func LoadGraphFromData(
	originStationID, destStationID int,
	stops []int) (
	*stationgraph.StationGraph,
	*algorithm.MonteCarloTree,
	chan *algorithm.MonteCarloTree,
	*skyline.RouteList) {

	origin := station.GetStationByID(originStationID)
	log.Printf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(destStationID)
	log.Printf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)

	// Get station graph based on the number of stops
	var graph *stationgraph.StationGraph
	var succeeded bool
	if len(stops) == 0 {
		graph, succeeded = stationgraph.BuildStationGraph(origin, dest)
	} else {
		log.Printf("use %d stops", len(stops))
		var stationStops []*station.Station
		for _, stop := range stops {
			stationStops = append(stationStops, station.GetStationByID(stop))
		}

		graph, succeeded = stationgraph.BuildGraphWithStops(origin, dest, stationStops)
	}

	if !succeeded {
		log.Fatal("graph build failed")
	}

	log.Printf("found %d routes\n", graph.CountRoutes())

	log.Printf("estimating station gains...")

	// TODO: Bugs in initualize estimators
	skyline.InitializeEstimators(graph)

	log.Printf("begin MCTS...")

	// Start MCTS
	tree := algorithm.BuildMonteCarloTree(graph)
	channel := make(chan *algorithm.MonteCarloTree, 1)
	list := skyline.BuildRouteList()
	go SearchSkylineRoutesWithMCTS(graph, tree, channel, list)

	return graph, tree, channel, list
}

// LoadLocationsFromCSVFile load all stations(over 10, 000) from .csv file
func LoadLocationsFromCSVFile(path string) {
	abspath, _ := filepath.Abs(path)
	log.Printf("Load station locations from %v...", abspath)

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	lines, err := csv.NewReader(fp).ReadAll()
	LocationIndex = make(map[int]*Location, len(lines))

	for _, line := range lines {
		location := Location{}
		location.ID, _ = strconv.Atoi(line[0])
		location.Lat, _ = strconv.ParseFloat(line[1], 64)
		location.Lon, _ = strconv.ParseFloat(line[2], 64)
		LocationIndex[location.ID] = &location
	}
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

// LoadBusRoutesFromCSVFile load bus routes from .csv file
func LoadBusRoutesFromCSVFile(path string) []*ExistBusRoute {
	abspath, _ := filepath.Abs(path)
	log.Printf("load data from %v...", abspath)

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	lines, err := csv.NewReader(fp).ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	retval := []*ExistBusRoute{}

	for _, line := range lines {
		var r ExistBusRoute
		r.ID, _ = strconv.Atoi(line[0])
		elements := strings.Split(line[1][1:len(line[1])-1], ",")

		r.Stations = []int{}
		for _, s := range elements {
			sid, _ := strconv.Atoi(s)
			r.Stations = append(r.Stations, sid)
		}
		// log.Printf("The array: %v", elements)
		retval = append(retval, &r)
	}

	return retval
}

// GetOSRMRouteResult Routing using OSRM
func GetOSRMRouteResult(points [][2]float64) []*Location {
	client := osrm.NewFromURL("https://osrm.zjvis.org")
	ctx, cancelFn := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancelFn()

	pts := []geo.Point{}
	for _, p := range points {
		pts = append(pts, *geo.NewPointFromLatLng(p[0], p[1]))
	}
	pointSet := geo.NewPointSet()
	pointSet.SetPoints(pts)

	resp, err := client.Route(ctx, osrm.RouteRequest{
		Profile:     "car",
		Coordinates: osrm.NewGeometryFromPointSet(*pointSet),
		Steps:       osrm.StepsTrue,
	})

	if err != nil {
		log.Printf("route failed: %v", err)
	}

	// log.Printf("routes are %+v", resp.Routes)
	return getRouteSteps(resp.Routes[0])
}

func getRouteSteps(r osrm.Route) []*Location {
	retval := []*Location{}
	for _, l := range r.Legs {
		for _, s := range l.Steps {
			for _, p := range s.Geometry.Path.PointSet {
				l := Location{Lon: p[0], Lat: p[1]}
				retval = append(retval, &l)
			}
		}
	}

	if len(retval) > 20 {
		newRetval := []*Location{}
		for i, s := range retval {
			if i%2 != 0 {
				newRetval = append(newRetval, s)
			}
		}
		retval = newRetval
	}
	return retval
}
