package algorithm

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func searchRouteRandomly(origin, dest *stationgraph.StationNode, g *stationgraph.StationGraph, list *skyline.RouteList, rng *rand.Rand) {
	curr := g.OriginNode
	route := []*station.Station{curr.S}

	// abandon this route candidate if num of retries drops to zero
	numRetries := 5

	for curr != g.DestNode && numRetries > 0 {
		if len(curr.Next) == 0 {
			return
		}

		f := make([]int, len(curr.Next))
		fsum := 0
		for i, n := range curr.Next {
			for _, s := range route {
				f[i] += skyline.Fmatrix[s.ID][n.S.ID]
			}
			fsum += f[i]
		}

		var next *stationgraph.StationNode
		if fsum == 0 {
			next = curr.Next[rng.Intn(len(curr.Next))]
		} else {
			target := rng.Float64()
			acc := float64(0)
			for i, fv := range f {
				acc += float64(fv) / float64(fsum)
				if acc >= target {
					next = curr.Next[i]
					break
				}
			}
		}

		if stationgraph.IsZigzagRoute(next.S, route) {
			numRetries--
		} else {
			numRetries = 5
			curr = next
			route = append(route, curr.S)
		}
	}

	if numRetries > 0 {
		list.Add(route, nil)
	}
}

// SearchSkylineRoutesWithPBS searches skyline routes with the PBS algorithm
func SearchSkylineRoutesWithPBS(g *stationgraph.StationGraph, iteration int, bps bool) *skyline.RouteList {
	var rg *stationgraph.StationGraph
	if bps {
		rg = g.GenerateReverseGraph()
	}

	list := skyline.BuildRouteList()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10*iteration; i++ {
		if list.NumAdds >= iteration {
			break
		}
		searchRouteRandomly(g.OriginNode, g.DestNode, g, list, rng)
		if bps {
			searchRouteRandomly(rg.OriginNode, rg.DestNode, rg, list, rng)
		}
	}

	return list
}

// SearchSkylineRoutesWithPBSInParallel searches skyline routes with the PBS algorithm in parallel
func SearchSkylineRoutesWithPBSInParallel(g *stationgraph.StationGraph, iteration int, bps bool) *skyline.RouteList {
	var wg sync.WaitGroup

	var rg *stationgraph.StationGraph
	if bps {
		rg = g.GenerateReverseGraph()
	}

	list := skyline.BuildRouteList()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	wg.Add(iteration)
	for i := 0; i < iteration; i++ {
		go func() {
			searchRouteRandomly(g.OriginNode, g.DestNode, g, list, rng)
			if bps {
				searchRouteRandomly(rg.OriginNode, rg.DestNode, rg, list, rng)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	log.Printf("found %d skyline routes\n", len(list.Routes))
	return list
}
