package stationgraph

import (
	"math"
	"math/big"
	"sort"
	"sync"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"

	log "github.com/sirupsen/logrus"
)

// StationNode declares a node in a directed station graph.
type StationNode struct {
	S    *station.Station
	Prev []*StationNode
	Next []*StationNode
	Hit  int
}

// StationGraph declares a directed station graph consisting a series of station nodes.
type StationGraph struct {
	OriginNode *StationNode
	DestNode   *StationNode
	GivenStops []*StationNode
	Nodes      []*StationNode
	NodesIndex map[int]int // map from station id to graph id
	Mutex      sync.Mutex
}

// Duplicate returns a duplicate of the given station graph
func (g *StationGraph) Duplicate() *StationGraph {
	ng := &StationGraph{NodesIndex: make(map[int]int)}
	for _, n := range g.Nodes {
		ng.addStationNode(&StationNode{S: n.S})
	}
	for _, n := range g.GivenStops {
		ng.AddStop(&StationNode{S: n.S})
	}

	ng.OriginNode = ng.Nodes[ng.NodesIndex[g.OriginNode.S.ID]]
	ng.DestNode = ng.Nodes[ng.NodesIndex[g.DestNode.S.ID]]
	for _, n := range ng.Nodes {
		for _, pn := range g.Nodes[g.NodesIndex[n.S.ID]].Prev {
			n.Prev = append(n.Prev, ng.Nodes[ng.NodesIndex[pn.S.ID]])
		}
		for _, nn := range g.Nodes[g.NodesIndex[n.S.ID]].Next {
			n.Next = append(n.Next, ng.Nodes[ng.NodesIndex[nn.S.ID]])
		}
	}

	for _, n := range ng.GivenStops {
		for _, pn := range g.Nodes[g.NodesIndex[n.S.ID]].Prev {
			n.Prev = append(n.Prev, ng.Nodes[ng.NodesIndex[pn.S.ID]])
		}
		for _, nn := range g.Nodes[g.NodesIndex[n.S.ID]].Next {
			n.Next = append(n.Next, ng.Nodes[ng.NodesIndex[nn.S.ID]])
		}
	}

	return ng
}

// FindNodeInList find a station node from a station node list. Not efficient. To be improved.
func FindNodeInList(s *StationNode, list []*StationNode) int {
	for idx, node := range list {
		if node.S.ID == s.S.ID {
			return idx
		}
	}
	return -1
}

// just swap and delete the last one
func remove(s []*StationNode, i int) []*StationNode {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// DropNode drop a node from the station graph
func (g *StationGraph) DropNode(sid int) *StationGraph {
	gid := g.NodesIndex[sid]
	node := g.Nodes[gid]
	for _, pn := range node.Prev {
		if i := FindNodeInList(node, pn.Next); i < 0 {
			log.Panic("Could not find the node in the Previous node's Next...")
		} else {
			pn.Next = remove(pn.Next, i)
		}
	}
	for _, nn := range node.Next {
		if i := FindNodeInList(node, nn.Prev); i < 0 {
			log.Panic("Could not find the node in the Next node's Prev...")
		} else {
			nn.Prev = remove(nn.Prev, i)
		}
	}
	ridx := g.NodesIndex[node.S.ID]
	// adjust index for removing node
	g.NodesIndex[g.Nodes[len(g.Nodes)-1].S.ID] = g.NodesIndex[node.S.ID]
	g.Nodes = remove(g.Nodes, ridx)

	// Do not remove index, not neccesary
	return g
}

// IsEqual asserts whether two graphs are equal
func (g *StationGraph) IsEqual(ng *StationGraph) bool {
	if len(g.Nodes) != len(ng.Nodes) {
		log.Error("GRAPH EQUAL ASSERTION FAILED: NODE COUNT NOT MATCH")
		return false
	}
	for _, n := range g.Nodes {
		var m *StationNode
		if p, exists := ng.NodesIndex[n.S.ID]; exists {
			m = ng.Nodes[p]
		} else {
			log.Error("GRAPH EQUAL ASSERTION FAILED: NODE NOT FOUND")
			return false
		}
		if len(n.Prev) != len(m.Prev) {
			log.Error("GRAPH EQUAL ASSERTION FAILED: PREV COUNT MISMATCH")
			return false
		}
		if len(n.Next) != len(m.Next) {
			log.Error("GRAPH EQUAL ASSERTION FAILED: NEXT COUNT MISMATCH")
			return false
		}
		for i := 0; i < len(n.Prev); i++ {
			if n.Prev[i].S != m.Prev[i].S {
				log.Error("GRAPH EQUAL ASSERTION FAILED: PREV STATION MISMATCH")
				return false
			}
		}
		for i := 0; i < len(n.Next); i++ {
			if n.Next[i].S != m.Next[i].S {
				log.Error("GRAPH EQUAL ASSERTION FAILED: NEXT STATION MISMATCH")
				return false
			}
		}
	}
	return true
}

func (g *StationGraph) addStationNode(n *StationNode) *StationNode {
	g.NodesIndex[n.S.ID] = len(g.Nodes)
	g.Nodes = append(g.Nodes, n)
	return n
}

func (g *StationGraph) setOriginStation(origin *station.Station) {
	g.OriginNode = &StationNode{S: origin}
	g.addStationNode(g.OriginNode)
}

// AddStop add a stop to the graph
func (g *StationGraph) AddStop(stop *StationNode) {
	g.GivenStops = append(g.GivenStops, stop)
}

func (g *StationGraph) setDestStation(dest *station.Station) {
	g.DestNode = &StationNode{S: dest}
	g.addStationNode(g.DestNode)
}

func (g *StationGraph) findPlausibleNeighborStations(s *station.Station) []*station.Station {
	var candidates []*station.Station
	for _, id := range s.Neighbors {
		next := station.GetStationByID(id)
		if IsMovingForward(next, s, g.OriginNode.S, g.DestNode.S) &&
			IsOriginFarther(next, s, g.OriginNode.S) &&
			IsDestinationCloser(next, s, g.DestNode.S) {
			candidates = append(candidates, next)
		}
	}
	return candidates
}

func (g *StationGraph) searchAdjacentStationNodes(node *StationNode, wg *sync.WaitGroup) {
	candidates := g.findPlausibleNeighborStations(node.S)

	g.Mutex.Lock()
	var nextSearchNodes []*StationNode
	for _, s := range candidates {
		var n *StationNode
		if id, exists := g.NodesIndex[s.ID]; !exists {
			n = g.addStationNode(&StationNode{S: s})
			nextSearchNodes = append(nextSearchNodes, n)
		} else {
			n = g.Nodes[id]
		}
		node.Next = append(node.Next, n)
		n.Prev = append(n.Prev, node)
	}
	g.Mutex.Unlock()

	wg.Add(len(nextSearchNodes))
	for _, n := range nextSearchNodes {
		go g.searchAdjacentStationNodes(n, wg)
	}
	wg.Done()
}

func (g *StationGraph) purgeUnreachableLinks(curr *StationNode, reachable map[int]bool) bool {
	if curr == g.DestNode {
		return true
	}

	if result, exists := reachable[curr.S.ID]; exists {
		return result
	}

	// eliminate loops if any
	reachable[curr.S.ID] = false

	var newNext []*StationNode
	for _, n := range curr.Next {
		r := g.purgeUnreachableLinks(n, reachable)
		if r {
			newNext = append(newNext, n)
		} else {
			n.Prev = nil
		}
	}

	curr.Next = newNext
	reachable[curr.S.ID] = len(newNext) != 0
	return len(newNext) != 0
}

func (g *StationGraph) countNodeRoutes(curr *StationNode, table map[int]*big.Int) *big.Int {
	if curr == g.DestNode {
		return big.NewInt(1)
	}
	if cnt, ok := table[curr.S.ID]; ok {
		return cnt
	}

	result := big.NewInt(0)
	for _, n := range curr.Next {
		result.Add(result, g.countNodeRoutes(n, table))
	}

	// log.Printf("node %d has %v routes to the destination", curr.S.ID, result)
	table[curr.S.ID] = result
	return result
}

// CountRoutes counts how many available routes are there in a station graph
func (g *StationGraph) CountRoutes() *big.Int {
	return g.countNodeRoutes(g.OriginNode, make(map[int]*big.Int))
}

// CountChoices counts how many choices for each node
func (g *StationGraph) CountChoices() {
	var choices []int
	for _, n := range g.Nodes {
		choices = append(choices, len(n.Next))
	}

	max := math.Inf(-1)
	min := math.Inf(1)
	total := 0.0

	for _, c := range choices {
		total += float64(c)
		min = math.Min(min, float64(c))
		max = math.Max(max, float64(c))
	}

	log.Printf("ks: total num nodes %d avg %.6f min %.1f max %.1f", len(choices), total/float64(len(choices)), min, max)
}

func (g *StationGraph) findNodeLoops(
	curr *StationNode,
	passedIDs []int,
	index map[int]int,
	visited map[int]bool,
) bool {
	if visited[curr.S.ID] {
		return false
	}
	if curr == g.DestNode {
		return false
	}

	visited[curr.S.ID] = true
	found := false
	for _, n := range curr.Next {
		if offset, ok := index[n.S.ID]; ok {
			log.Warnf("Loop detected %v", passedIDs[offset:])
			found = true
		} else {
			index[n.S.ID] = len(passedIDs)
			newPassIDs := append([]int(nil), passedIDs...)
			newPassIDs = append(newPassIDs, n.S.ID)
			f := g.findNodeLoops(n, newPassIDs, index, visited)
			delete(index, n.S.ID)
			found = found || f
		}
	}
	return found
}

// HasLoop detects whether there is a loop in the graph
func (g *StationGraph) HasLoop() bool {
	return g.findNodeLoops(g.OriginNode, []int{}, make(map[int]int), make(map[int]bool))
}

func (g *StationGraph) reverseNode(n *StationNode, rg *StationGraph, visited map[int]bool) {
	if visited[n.S.ID] || n == g.DestNode {
		return
	}

	var rn *StationNode
	if i, exists := rg.NodesIndex[n.S.ID]; !exists {
		rn = rg.addStationNode(&StationNode{S: n.S})
	} else {
		rn = rg.Nodes[i]
	}

	visited[n.S.ID] = true
	for _, next := range n.Next {
		if i, exists := rg.NodesIndex[next.S.ID]; exists {
			rg.Nodes[i].Next = append(rg.Nodes[i].Next, rn)
			rn.Prev = append(rn.Prev, rg.Nodes[i])
		} else {
			nn := rg.addStationNode(&StationNode{S: next.S})
			nn.Next = append(nn.Next, rn)
			rn.Prev = append(rn.Prev, nn)
		}
		g.reverseNode(next, rg, visited)
	}
}

// GenerateReverseGraph generates a graph where every link is reversed
func (g *StationGraph) GenerateReverseGraph() *StationGraph {
	rg := &StationGraph{NodesIndex: make(map[int]int)}
	rg.setOriginStation(g.DestNode.S)
	rg.setDestStation(g.OriginNode.S)

	g.reverseNode(g.OriginNode, rg, make(map[int]bool))
	return rg
}

// SortTopo returns topologically sorted stations
func (g *StationGraph) SortTopo() []*StationNode {
	var sorted []*StationNode

	Q := []*StationNode{g.OriginNode}
	indegree := make(map[int]int)
	for i := 0; i < len(Q); i++ {
		for _, ns := range Q[i].Next {
			if len(ns.Prev) == 1 {
				Q = append(Q, ns)
			} else if ind, exists := indegree[ns.S.ID]; exists {
				if ind == 1 {
					Q = append(Q, ns)
				}
				indegree[ns.S.ID]--
			} else {
				indegree[ns.S.ID] = len(ns.Prev) - 1
			}
		}
		sorted = append(sorted, Q[i])
	}
	return sorted
}

// BuildStationGraph builds a station graph from origin to destination stations
func BuildStationGraph(origin, dest *station.Station) (*StationGraph, bool) {
	g := &StationGraph{NodesIndex: make(map[int]int)}

	// create the first & last nodes
	g.setOriginStation(origin)
	g.setDestStation(dest)

	var wg sync.WaitGroup
	wg.Add(1)
	g.searchAdjacentStationNodes(g.OriginNode, &wg)
	wg.Wait()

	reachable := g.purgeUnreachableLinks(g.OriginNode, make(map[int]bool))
	if !reachable {
		log.Warn("destination is not reachable!")
	}

	loop := g.HasLoop()
	if !loop {
		log.Info("loop check passed")
	}

	return g, reachable && !loop
}

// BuildSubGraph builds a station graph based on stops and node/link dropping
func BuildSubGraph(g *StationGraph) *StationGraph {
	g.AddStop(g.DestNode)

	ng := g.Duplicate()
	origin := ng.OriginNode
	sortedNodes := ng.SortTopo()

	stationIndexMap := make(map[int]int)
	for i, s := range sortedNodes {
		stationIndexMap[s.S.ID] = i
	}
	// sort stops by topo
	sort.Slice(ng.GivenStops, func(i, j int) bool {
		i1 := stationIndexMap[ng.GivenStops[i].S.ID]
		j1 := stationIndexMap[ng.GivenStops[j].S.ID]
		return i1 < j1
		// return stationIndexMap[ng.GivenStops[i].S.ID] < stationIndexMap[ng.GivenStops[j].S.ID]
	})

	for _, stop := range ng.GivenStops {
		originID := stationIndexMap[origin.S.ID]
		destinateID := stationIndexMap[stop.S.ID]

		for index := originID; index < destinateID; index++ {
			node := sortedNodes[index]
			newNext := []*StationNode{}

			for _, nextNode := range node.Next {
				if stationIndexMap[nextNode.S.ID] > destinateID {
					continue
				}
				newNext = append(newNext, nextNode)
			}

			node.Next = make([]*StationNode, len(newNext))
			copy(node.Next, newNext)
		}

		for index := destinateID - 1; index > originID; index-- {
			node := sortedNodes[index]
			if len(node.Next) == 0 {
				ng = ng.DropNode(node.S.ID)
			}
		}

		origin = stop
	}

	return ng
}

// BuildGraphWithStops build graphs between every two stops and
func BuildGraphWithStops(origin, dest *station.Station, stops []*station.Station) (*StationGraph, bool) {

	g := &StationGraph{NodesIndex: make(map[int]int)}

	stops = append(stops, dest)
	// TODO find a good path from origin to dest
	// Now assume the order is given by users

	o := origin // origin in every sub network
	for _, stop := range stops {
		subgraph, succeed := BuildStationGraph(o, stop)
		if !succeed {
			return g, succeed
		}

		// combine the subgraph with the big graph
		if len(g.Nodes) == 0 {
			g = subgraph
			o = stop
			continue
		}

		g.Nodes[g.NodesIndex[g.DestNode.S.ID]].Next = append(g.Nodes[g.NodesIndex[g.DestNode.S.ID]].Next, subgraph.OriginNode.Next...)
		// move the origin in the sub graph to the 1st position
		if subgraph.NodesIndex[subgraph.OriginNode.S.ID] != 0 {
			oi := subgraph.NodesIndex[subgraph.OriginNode.S.ID]
			subgraph.Nodes[0], subgraph.Nodes[oi] = subgraph.Nodes[oi], subgraph.Nodes[0]
			subgraph.NodesIndex[subgraph.Nodes[oi].S.ID] = oi
			subgraph.NodesIndex[subgraph.Nodes[0].S.ID] = 0
		}
		// change node index map
		glen := len(g.Nodes)
		for key, value := range subgraph.NodesIndex {
			if value == 0 {
				continue
			}
			g.NodesIndex[key] = value + glen
		}
		g.Nodes = append(g.Nodes, subgraph.Nodes[1:]...)
		g.GivenStops = append(g.GivenStops, g.Nodes[g.NodesIndex[o.ID]])
		o = stop
		g.DestNode = subgraph.DestNode
	}

	return g, true
}

// type StationGraph struct {
// 	OriginNode *StationNode
// 	DestNode   *StationNode
// 	GivenStops []*StationNode
// 	Nodes      []*StationNode
// 	NodesIndex map[int]int // map from station id to graph id
// 	Mutex      sync.Mutex
// }
