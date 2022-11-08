package algorithm

import (
	"log"
	"math"
	"math/rand"
	"sort"
	"time"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// MonteCarloTreeNodeValues defines the values of a node in a monte carlo search tree
type MonteCarloTreeNodeValues struct {
	T     float64
	Num   float64
	Count int

	W           float64
	NumSkylines int

	UValue float64
}

// MonteCarloTreeNode defines a node in a monte carlo search tree
type MonteCarloTreeNode struct {
	Values           MonteCarloTreeNodeValues
	N                int
	Actions          []*MonteCarloTreeNode
	AvailableActions []*stationgraph.StationNode
	SNode            *stationgraph.StationNode
	Parent           *MonteCarloTreeNode
}

// MonteCarloTreeRandomHeuristic defines what heuristic should be used when choosing the next station
type MonteCarloTreeRandomHeuristic int

const (
	// MonteCarloTreeRandomHeuristicUniformRandom uses the uniform random
	MonteCarloTreeRandomHeuristicUniformRandom MonteCarloTreeRandomHeuristic = iota
	// MonteCarloTreeRandomHeuristicPBS uses the strategy that PBS algorithm uses
	MonteCarloTreeRandomHeuristicPBS
	// MonteCarloTreeRandomHeuristicEstimation uses the average estimations for all criteria
	MonteCarloTreeRandomHeuristicEstimation
)

// MonteCarloTree defines a monte carlo search tree
type MonteCarloTree struct {
	Done            bool
	Root            *MonteCarloTreeNode
	Graph           *stationgraph.StationGraph
	RandomHeuristic MonteCarloTreeRandomHeuristic
	Alpha           float64
	Rng             *rand.Rand
	Nodes           []*MonteCarloTreeNode
}

// ComputeUCT c controls the bias between the credit and the number of visited times
func (tree *MonteCarloTree) ComputeUCT(n *MonteCarloTreeNode, c float64) float64 {
	return float64(n.Values.W)/float64(n.N) + c*math.Sqrt(math.Log(float64(n.Parent.N))/float64(n.N))
}

func (tree *MonteCarloTree) choose(n *MonteCarloTreeNode, c float64) (*MonteCarloTreeNode, bool) {
	if len(n.Actions) == 0 && len(n.AvailableActions) == 0 {
		return nil, false
	}

	// if the node n is fully ded
	for len(n.AvailableActions) == 0 {
		n.N++
		n.SNode.Hit++

		// default to the only action
		if len(n.Actions) == 1 {
			n = n.Actions[0]
		} else {
			var maxuct float64
			var action *MonteCarloTreeNode

			// find an action with maximum UCT
			for _, a := range n.Actions {
				uct := tree.ComputeUCT(a, c)
				if action == nil || uct > maxuct {
					action = a
					maxuct = uct
				}
			}

			n = action
		}
	}

	// about to expand the node n
	n.N++
	n.SNode.Hit++
	return n, true
}

// GetRouteToNode returns a route to a specified node
func (tree *MonteCarloTree) GetRouteToNode(n *MonteCarloTreeNode) []*station.Station {
	return tree.GetRouteFromPath(tree.GetPathToNode(n))
}

// GetPathToNode returns a path to a specified node
func (tree *MonteCarloTree) GetPathToNode(n *MonteCarloTreeNode) []*stationgraph.StationNode {
	var r []*stationgraph.StationNode
	for n != nil {
		r = append(r, n.SNode)
		n = n.Parent
	}
	for left, right := 0, len(r)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = r[right], r[left]
	}
	return r
}

// GetRouteFromPath returns the route that corresponds to the given path
func (tree *MonteCarloTree) GetRouteFromPath(path []*stationgraph.StationNode) []*station.Station {
	r := make([]*station.Station, len(path))
	for i := range path {
		r[i] = path[i].S
	}
	return r
}

func (tree *MonteCarloTree) removeAvailableActionByIndex(n *MonteCarloTreeNode, i int) {
	if len(n.AvailableActions) == 1 {
		n.AvailableActions = nil
	} else {
		n.AvailableActions[i], n.AvailableActions[len(n.AvailableActions)-1] = n.AvailableActions[len(n.AvailableActions)-1], n.AvailableActions[i]
		n.AvailableActions[len(n.AvailableActions)-1] = nil
		n.AvailableActions = n.AvailableActions[:len(n.AvailableActions)-1]
	}
}

func (tree *MonteCarloTree) removeAction(n *MonteCarloTreeNode, action *MonteCarloTreeNode) {
	if len(n.Actions) == 1 {
		n.Actions = nil
		return
	}
	for i := range n.Actions {
		if n.Actions[i] == action {
			n.Actions[i], n.Actions[len(n.Actions)-1] = n.Actions[len(n.Actions)-1], n.Actions[i]
			n.Actions[len(n.Actions)-1] = nil
			n.Actions = n.Actions[:len(n.Actions)-1]
			return
		}
	}
}

func (tree *MonteCarloTree) cleanInvalidNode(n *MonteCarloTreeNode) *MonteCarloTreeNode {
	for n != nil {
		if len(n.Actions) == 0 && len(n.AvailableActions) == 0 {
			tree.removeAction(n.Parent, n)
			n = n.Parent
		} else {
			break
		}
	}
	return n
}

// ComputeChoiceWeights returns the weights for choices based on the heuristics
func (tree *MonteCarloTree) ComputeChoiceWeights(choices []*stationgraph.StationNode, path []*stationgraph.StationNode) []float64 {
	weights := make([]float64, len(choices))
	switch tree.RandomHeuristic {
	case MonteCarloTreeRandomHeuristicPBS:
		for i, c := range choices {
			for _, s := range path {
				weights[i] += float64(skyline.Fmatrix[s.S.ID][c.S.ID])
			}
		}
	case MonteCarloTreeRandomHeuristicEstimation:
		values := make([][]float64, len(choices))
		min := make([]float64, len(skyline.RouteCriteria))
		max := make([]float64, len(skyline.RouteCriteria))
		for k := range skyline.RouteCriteria {
			min[k] = math.Inf(1)
			max[k] = math.Inf(-1)
		}
		for i, s := range choices {
			values[i] = make([]float64, len(skyline.RouteCriteria))
			for k, c := range skyline.RouteCriteria {
				values[i][k] = c.PredictStationGain(path, s)
				if values[i][k] < min[k] {
					min[k] = values[i][k]
				}
				if values[i][k] > max[k] {
					max[k] = values[i][k]
				}
			}
		}
		weights = make([]float64, len(choices))
		for i := range choices {
			for k, c := range skyline.RouteCriteria {
				weights[i] += c.Normalize(values[i][k], min[k], max[k])
			}
			weights[i] = math.Pow(weights[i]/float64(len(skyline.RouteCriteria)), tree.Alpha)
			// weights[i] = values[i][1] // math.Sqrt(weights[i])
		}
	}
	return weights
}

func (tree *MonteCarloTree) getRandomStationNode(
	choices []*stationgraph.StationNode,
	weights []float64,
	path []*stationgraph.StationNode,
) (*stationgraph.StationNode, []*stationgraph.StationNode, []float64) {
	if choices == nil {
		return nil, nil, nil
	}
	if len(choices) == 1 {
		return choices[0], nil, nil
	}

	if weights == nil {
		weights = tree.ComputeChoiceWeights(choices, path)
	}

	wsum := 0.0
	for _, w := range weights {
		wsum += w
	}

	route := tree.GetRouteFromPath(path)
	for len(choices) > 0 {
		var k int
		if wsum < 1e-7 {
			k = tree.Rng.Intn(len(choices))
		} else {
			p := tree.Rng.Float64() * wsum
			// log.Printf("wsum=%f p=%f", wsum, p)
			for i, v := range weights {
				if v > p {
					k = i
					break
				}
				p -= v
			}
		}

		result := choices[k]
		choices[k], choices[len(choices)-1] = choices[len(choices)-1], choices[k]
		choices = choices[:len(choices)-1]
		wsum -= weights[k]
		weights[k], weights[len(weights)-1] = weights[len(weights)-1], weights[k]
		weights = weights[:len(weights)-1]

		if !stationgraph.IsZigzagRoute(result.S, route) {
			return result, choices, weights
		}
	}

	return nil, nil, nil
}

func (tree *MonteCarloTree) expand(n *MonteCarloTreeNode) (*MonteCarloTreeNode, bool) {
	var nextAvailableAction *stationgraph.StationNode
	path := tree.GetPathToNode(n)

	// find an available action that satisfies the zigzag constraint
	var weights []float64
	for {
		nextAvailableAction, n.AvailableActions, weights = tree.getRandomStationNode(n.AvailableActions, weights, path)

		// only when the whole search space is covered
		if nextAvailableAction == nil {
			return tree.cleanInvalidNode(n), false
		}
		if nextAvailableAction != tree.Graph.DestNode {
			break
		}
	}

	nextAction := makeMonteCarloTreeNode(nextAvailableAction, n)
	tree.Nodes = append(tree.Nodes, nextAction)
	n.Actions = append(n.Actions, nextAction)
	return nextAction, true
}

func (tree *MonteCarloTree) simulate(n *MonteCarloTreeNode, list *skyline.RouteList, root *MonteCarloTreeNode) bool {
	n.N++
	s := n.SNode
	path := tree.GetPathToNode(n)

	//TODO the limitation of the time complexity
	for s != tree.Graph.DestNode {
		s.Hit++
		n, _, _ := tree.getRandomStationNode(append([]*stationgraph.StationNode{}, s.Next...), nil, path)

		if n == nil {
			return false
		}

		path = append(path, n)
		s = n
	}

	// The path contains every node from the root to the leaf
	added, _ := list.Add(tree.GetRouteFromPath(path), func() {
		n.Values.NumSkylines--
		tree.backpropagate(n, root)
	})

	if added {
		n.Values.NumSkylines++
		tree.backpropagate(n, root)
	}

	// n.Values.T = (n.Values.T*float64(n.Values.Count) + values.T) / float64(n.Values.Count+1)
	// n.Values.Num = (n.Values.Num*float64(n.Values.Count) + values.Num) / float64(n.Values.Count+1)
	// n.Values.Count++

	return true
}

func (tree *MonteCarloTree) backpropagate(n *MonteCarloTreeNode, root *MonteCarloTreeNode) {
	for n != root.Parent {
		n.Values.W = float64(n.Values.NumSkylines)
		for _, a := range n.Actions {
			n.Values.W += a.Values.W
		}
		n = n.Parent
	}
}

func (tree *MonteCarloTree) checkIntegrity(n *MonteCarloTreeNode) {
	if len(n.Actions) == 0 && len(n.AvailableActions) == 0 {
		log.Panic("actions should not be nil")
	}
	for _, a := range n.Actions {
		tree.checkIntegrity(a)
	}
}

// ExploreOnce performs one walk in the monte-carlo tree
func (tree *MonteCarloTree) ExploreOnce(root *MonteCarloTreeNode, list *skyline.RouteList, c float64) bool {
	// tree.checkIntegrity(root)

	// log.Printf("choosing a node to expand...")
	n, succeeded := tree.choose(root, c)
	if !succeeded {
		// log.Println("warn: tree has been completely explored")
		return false
	}

	// log.Printf("attempting to expand node %d...", n.SNode.S.ID)
	e, succeeded := tree.expand(n)
	if !succeeded {
		// log.Println("info: this node cannot be expanded, good luck next time")
		return true
	}

	// log.Printf("simulating random walk...")
	if succeeded := tree.simulate(e, list, root); !succeeded {
		// log.Println("info: this path cannot reach the destination, good luck next time")
	}

	return true
}

// Print prints a monte carlo tree
func (tree *MonteCarloTree) Print(n *MonteCarloTreeNode, depth int, c float64) {
	// log.Printf("ID: %d; Value: { T: %.3f, Num: %.3f = W: %.3f }, N: %d, AA: %d, Actions: [",
	// 	n.SNode.S.ID, n.Values.T, n.Values.Num, n.Values.W, n.N, len(n.AvailableActions))
	log.Printf("ID: %d; Value: { NS: %d = W: %.3f }, N: %d, AA: %d, Actions: [",
		n.SNode.S.ID, n.Values.NumSkylines, n.Values.W, n.N, len(n.AvailableActions))
	for _, a := range n.Actions {
		// log.Printf("  ID: %d, Value: { T: %.3f, Num: %.3f = W: %.3f }, N: %d, AA: %d, UCT: %.3f",
		// 	a.SNode.S.ID, a.Values.T, a.Values.Num, a.Values.W, a.N, len(a.AvailableActions), tree.ComputeUCT(a))
		log.Printf("  ID: %d, Value: { NS: %d = W: %.3f }, N: %d, AA: %d, UCT: %.3f",
			a.SNode.S.ID, a.Values.NumSkylines, a.Values.W, a.N, len(a.AvailableActions), tree.ComputeUCT(a, c))
	}
	log.Printf("]")
	if depth > 1 {
		for _, a := range n.Actions {
			tree.Print(a, depth-1, c)
		}
	}
}

func makeMonteCarloTreeNode(n *stationgraph.StationNode, parent *MonteCarloTreeNode) *MonteCarloTreeNode {
	return &MonteCarloTreeNode{
		Values:           MonteCarloTreeNodeValues{T: 999999},
		N:                0,
		SNode:            n,
		AvailableActions: append([]*stationgraph.StationNode(nil), n.Next...),
		Parent:           parent,
	}
}

// BuildMonteCarloTree builds a blank monte carlo tree with the given station graph
func BuildMonteCarloTree(g *stationgraph.StationGraph) *MonteCarloTree {
	tree := &MonteCarloTree{
		Root:  makeMonteCarloTreeNode(g.OriginNode, nil),
		Graph: g,
		Nodes: []*MonteCarloTreeNode{},
	}
	// The root is meaning less
	// tree.Nodes = append(tree.Nodes, tree.Root)
	return tree
}

// MultiStopsSearchRoutesWithMCTS finds skyline routes with MCTS algorithm and multiple stops setting
func MultiStopsSearchRoutesWithMCTS(
	g *stationgraph.StationGraph,
	maxPoolSize, maxNumIter, minPoolSize, minNumIter, numSteps int,
	poolSizeDecay, numIterDecay float64,
	c float64,
	heuristic MonteCarloTreeRandomHeuristic,
	alpha float64,
) *skyline.RouteList {
	list := skyline.BuildRouteList()
	return list
}

// SearchSkylineRoutesWithMCTS finds skyline routes with MCTS algorithm
func SearchSkylineRoutesWithMCTS(
	g *stationgraph.StationGraph,
	maxPoolSize, maxNumIter, minPoolSize, minNumIter, numSteps int,
	poolSizeDecay, numIterDecay float64,
	c float64,
	heuristic MonteCarloTreeRandomHeuristic,
	alpha float64,
) *skyline.RouteList {
	list := skyline.BuildRouteList()
	tree := BuildMonteCarloTree(g)
	tree.RandomHeuristic = heuristic
	tree.Alpha = alpha
	tree.Rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	poolsize := float64(maxPoolSize) + 1e-7
	numiter := float64(maxNumIter) + 1e-7
	pool := []*MonteCarloTreeNode{tree.Root}
	for k := 0; k < numSteps; k++ {
		for _, n := range pool {
			maxi := int(numiter)
			for i := 0; i < maxi; i++ {
				// log.Printf("MCTS SEARCHING %d - %d", k, i)
				if succeeded := tree.ExploreOnce(n, list, c); !succeeded {
					break
				}
			}
		}

		if k != numSteps-1 {
			var newpool []*MonteCarloTreeNode
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
			if int(poolsize*poolSizeDecay) >= minPoolSize {
				poolsize *= poolSizeDecay
			}
			if int(numiter*numIterDecay) >= minNumIter {
				numiter *= numIterDecay
			}
		}
	}

	return list
}
