package skyline

import (
	"math/big"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// RouteFlowCriterion defines the flow criterion for the routes
type RouteFlowCriterion struct {
	sortedStations  []*stationgraph.StationNode
	stationIndexMap map[int]int
	numRoutes       map[int]map[int]*big.Int
	estimatedFlows  [][]float64
}

// Name returns the name of criterion
func (c *RouteFlowCriterion) Name() string {
	return "Num"
}

// Evaluate returns the computed criterion value for a route
func (c *RouteFlowCriterion) Evaluate(r []*station.Station) float64 {
	var result float64
	for i := 0; i < len(r); i++ {
		for j := i + 1; j < len(r); j++ {
			result += float64(Fmatrix[r[i].ID][r[j].ID])
		}
	}
	return result
}

// GreaterThan returns whether the criterion a is larger than b
func (c *RouteFlowCriterion) GreaterThan(a, b float64) bool {
	return a >= b
}

// Normalize returns the normalized criterion value between 0 and 1
func (c *RouteFlowCriterion) Normalize(v, min, max float64) float64 {
	return (v-min)/(max-min)*0.95 + 0.05
}

// InitEstimator initializes the criterion
func (c *RouteFlowCriterion) InitEstimator(sortedStations []*stationgraph.StationNode, stationIndexMap map[int]int) {
	c.sortedStations = sortedStations
	c.stationIndexMap = stationIndexMap

	c.numRoutes = make(map[int]map[int]*big.Int)
	for i := 0; i < len(sortedStations); i++ {
		c.numRoutes[i] = make(map[int]*big.Int)
	}
	for i := 0; i < len(sortedStations); i++ {
		count := make([]*big.Int, len(sortedStations))
		visited := make([]bool, len(sortedStations))

		for j := 0; j < i; j++ {
			count[j] = big.NewInt(0)
		}
		count[i] = big.NewInt(1)
		c.numRoutes[i][i] = count[i]
		visited[i] = true

		for j := i - 1; j >= 0; j-- {
			for _, ns := range sortedStations[j].Next {
				nsp := stationIndexMap[ns.S.ID]
				if nsp <= i && visited[nsp] {
					count[j].Add(count[j], count[nsp])
					visited[j] = true
				}
			}
			c.numRoutes[j][i] = count[j]
		}
	}

	d := len(c.sortedStations) - 1
	c.estimatedFlows = make([][]float64, len(c.sortedStations))
	c.estimatedFlows[d] = make([]float64, len(c.sortedStations))
	for i := 0; i < len(c.sortedStations)-1; i++ {
		c.estimatedFlows[i] = make([]float64, len(c.sortedStations))
		N := big.NewFloat(0).SetInt(c.numRoutes[i][d])

		sum := big.NewFloat(0)
		for p := i; p < len(c.sortedStations)-1; p++ {
			for q := p + 1; q < len(c.sortedStations); q++ {
				m := big.NewInt(0)
				m.Mul(c.numRoutes[i][p], c.numRoutes[p][q])
				m.Mul(m, c.numRoutes[q][d])
				mf := big.NewFloat(0).SetInt(m)
				flow := big.NewFloat(float64(Fmatrix[c.sortedStations[p].S.ID][c.sortedStations[q].S.ID]))
				sum.Add(sum, mf.Mul(mf, flow))
			}
		}
		avg, _ := sum.Quo(sum, N).Float64()
		c.estimatedFlows[i][i] = avg

		for p := 0; p < i; p++ {
			sum := big.NewFloat(0)
			for q := i; q < len(c.sortedStations); q++ {
				m := big.NewInt(0)
				m.Mul(c.numRoutes[i][q], c.numRoutes[q][d])
				mf := big.NewFloat(0).SetInt(m)
				flow := big.NewFloat(float64(Fmatrix[c.sortedStations[p].S.ID][c.sortedStations[q].S.ID]))
				sum.Add(sum, mf.Mul(mf, flow))
			}
			avg, _ := sum.Quo(sum, N).Float64()
			c.estimatedFlows[i][p] = avg
		}
		// log.Printf("Estimated average flows for station %d: %f", c.sortedStations[i].S.ID, avg)
	}
}

// PredictStationGain returns the gain for choosing a station while a route prefix is known
func (c *RouteFlowCriterion) PredictStationGain(path []*stationgraph.StationNode, choice *stationgraph.StationNode) float64 {
	// selected := make([]bool, len(c.sortedStations))
	var flow float64
	k := c.stationIndexMap[choice.S.ID]
	for _, s := range path {
		flow += c.estimatedFlows[k][c.stationIndexMap[s.S.ID]]
	}

	// log.Printf("flow predict for %d: %d %f", choice.S.ID, flow, c.estimatedFlows[c.stationIndexMap[choice.S.ID]])

	return float64(flow) + c.estimatedFlows[k][k]

	// fp := big.NewInt(0)
	// d := len(c.sortedStations) - 1
	// for p := 0; p < len(c.sortedStations)-1; p++ {
	// 	if p < k && !selected[p] {
	// 		continue
	// 	}

	// 	q := p + 1
	// 	if q <= k {
	// 		q = k + 1
	// 	}
	// 	for ; q < len(c.sortedStations); q++ {
	// 		m := big.NewInt(0)
	// 		switch {
	// 		case p <= k:
	// 			m.Mul(c.numRoutes[k][q], c.numRoutes[q][d])
	// 		case p > k:
	// 			m.Mul(c.numRoutes[k][p], c.numRoutes[p][q])
	// 			m.Mul(m, c.numRoutes[q][d])
	// 		}
	// 		m.Mul(m, big.NewInt(int64(Fmatrix[c.sortedStations[p].S.ID][c.sortedStations[q].S.ID])))
	// 		fp.Add(fp, m)
	// 	}
	// }
	// fpf := big.NewFloat(0).SetInt(fp)
	// predicted, _ := fpf.Quo(fpf, big.NewFloat(0).SetInt(c.numRoutes[k][d])).Float64()
	// return float64(flow) + predicted
}

// EstimateStationGain sets the gain for the specific criteria at a station
// func (c *RouteFlowCriterion) EstimateStationGain(
// 	s *stationgraph.StationNode,
// 	sortedStations []*stationgraph.StationNode,
// 	stationIndexMap map[int]int,
// ) float64 {
// 	flow := big.NewFloat(0)
// 	i := stationIndexMap[s.S.ID]
// 	o := 0
// 	d := len(sortedStations) - 1

// 	N := big.NewInt(0)
// 	for p := i; p < len(sortedStations); p++ {
// 		for q := p + 1; q < len(sortedStations); q++ {
// 			Fpq := big.NewFloat(float64(Fmatrix[sortedStations[p].S.ID][sortedStations[q].S.ID]))
// 			Npqi := big.NewInt(0)
// 			// switch {
// 			// case q <= i:
// 			// 	Npqi.Mul(c.numRoutes[o][p], c.numRoutes[p][q])
// 			// 	Npqi.Mul(Npqi, c.numRoutes[q][i])
// 			// 	Npqi.Mul(Npqi, c.numRoutes[i][d])
// 			// case q > i && p <= i:
// 			// 	Npqi.Mul(c.numRoutes[o][p], c.numRoutes[p][i])
// 			// 	Npqi.Mul(Npqi, c.numRoutes[i][q])
// 			// 	Npqi.Mul(Npqi, c.numRoutes[q][d])
// 			// case p > i:
// 			Npqi.Mul(c.numRoutes[o][i], c.numRoutes[i][p])
// 			Npqi.Mul(Npqi, c.numRoutes[p][q])
// 			Npqi.Mul(Npqi, c.numRoutes[q][d])
// 			// }
// 			flow.Add(flow, Fpq.Mul(Fpq, (&big.Float{}).SetInt(Npqi)))
// 		}
// 	}

// 	N.Mul(c.numRoutes[o][i], c.numRoutes[i][d])
// 	wf, _ := flow.Quo(flow, (&big.Float{}).SetInt(N)).Float64()
// 	return wf
// }
