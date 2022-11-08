package algorithm

import (
	"log"
	"sync"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func searchAllRoutesFromNode(curr *stationgraph.StationNode, r []*station.Station, g *stationgraph.StationGraph, list *skyline.RouteList, wg *sync.WaitGroup, limit chan struct{}) {
	for _, n := range curr.Next {
		if stationgraph.IsZigzagRoute(n.S, r) {
			continue
		}

		nr := append(r, n.S)
		if n == g.DestNode {
			list.Add(nr, nil)
		} else {
			pn := n
			wg.Add(1)
			go func() {
				limit <- struct{}{}
				searchAllRoutesFromNode(pn, nr, g, list, wg, limit)
				<-limit
			}()
		}
	}
	wg.Done()
}

// SearchSkylineRoutesBruteForce searches skyline routes
func SearchSkylineRoutesBruteForce(g *stationgraph.StationGraph) *skyline.RouteList {
	var wg sync.WaitGroup

	limit := make(chan struct{}, 20)
	list := skyline.BuildRouteList()

	wg.Add(1)
	searchAllRoutesFromNode(g.OriginNode, []*station.Station{g.OriginNode.S}, g, list, &wg, limit)
	wg.Wait()

	log.Printf("found %d skyline routes\n", len(list.Routes))

	return list
}
