package skyline

import (
	"math/big"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// RouteDirectnessCriterion defines the directness criterion for the routes
type RouteDirectnessCriterion struct {
	sortedStations  []*stationgraph.StationNode
	stationIndexMap map[int]int
	numRoutes       [][]*big.Float
	partialResults  []*big.Float
	// transit route distance divid road distance
	estimatedDirectness    [][]float64
	estimatedTransitLength [][]float64
}

// Name returns the name of criterion
func (c *RouteDirectnessCriterion) Name() string {
	return "Directness"
}

// Evaluate returns the computed criterion value for a route
func (c *RouteDirectnessCriterion) Evaluate(r []*station.Station) float64 {
	dist := make([]float64, len(r))
	dist[0] = 0
	for i := 1; i < len(r); i++ {
		dist[i] = dist[i-1] + Dmatrix[r[i].ID][r[i-1].ID]
	}

	var result float64
	var count int
	for i := 0; i < len(r); i++ {
		for j := i + 1; j < len(r); j++ {
			result += (dist[j] - dist[i]) / Dmatrix[r[i].ID][r[j].ID]
			count++
		}
	}
	return result / float64(count)
}

// GreaterThan returns whether the criterion a is larger than b
func (c *RouteDirectnessCriterion) GreaterThan(a, b float64) bool {
	return a <= b
}

// Normalize returns the normalized criterion value between 0 and 1
func (c *RouteDirectnessCriterion) Normalize(v, min, max float64) float64 {
	return (max-v)/(max-min)*0.95 + 0.05
}

// InitEstimator initializes the criterion
func (c *RouteDirectnessCriterion) InitEstimator(sortedStations []*stationgraph.StationNode, stationIndexMap map[int]int) {
	c.sortedStations = sortedStations
	c.stationIndexMap = stationIndexMap

	c.estimatedTransitLength = make([][]float64, len(sortedStations))
	c.estimatedDirectness = make([][]float64, len(sortedStations))

	c.numRoutes = make([][]*big.Float, len(sortedStations))
	c.partialResults = make([]*big.Float, len(sortedStations))
	for i := 0; i < len(sortedStations); i++ {
		c.estimatedTransitLength[i] = make([]float64, len(sortedStations))
		c.estimatedDirectness[i] = make([]float64, len(sortedStations))

		c.numRoutes[i] = make([]*big.Float, len(sortedStations))
		c.partialResults[i] = big.NewFloat(0)
		for j := 0; j < len(sortedStations); j++ {
			c.numRoutes[i][j] = big.NewFloat(0)
		}
	}

	for j := len(sortedStations) - 1; j >= 0; j-- {
		accumulatedLength := make([]*big.Float, len(sortedStations))
		accumulatedLength[j] = big.NewFloat(0)
		c.numRoutes[j][j].SetFloat64(1)

		for i := j - 1; i >= 0; i-- {
			accumulatedLength[i] = big.NewFloat(0)

			for _, s := range sortedStations[i].Next {
				npos := stationIndexMap[s.S.ID]
				if npos > j {
					continue
				}

				multipliedLength := big.NewFloat(Dmatrix[sortedStations[i].S.ID][sortedStations[npos].S.ID])
				multipliedLength.Mul(multipliedLength, c.numRoutes[npos][j])
				accumulatedLength[i].Add(accumulatedLength[i], accumulatedLength[npos])
				accumulatedLength[i].Add(accumulatedLength[i], multipliedLength)
				c.numRoutes[i][j].Add(c.numRoutes[i][j], c.numRoutes[npos][j])
			}

			numRoute, _ := c.numRoutes[i][j].Int64()
			if numRoute != 0 {
				total := new(big.Float).Set(accumulatedLength[i])
				c.estimatedTransitLength[i][j], _ = total.Quo(total, c.numRoutes[i][j]).Float64()
			} else {
				c.estimatedTransitLength[i][j] = Dmatrix[sortedStations[i].S.ID][sortedStations[j].S.ID]
			}
		}
	}

	d := len(c.sortedStations) - 1
	// precompute the subspace grade

	for k := 0; k < len(c.sortedStations); k++ {
		for v := k; v < len(c.sortedStations); v++ {
			u1 := k
			k1 := v
			if k1 >= u1+1 {
				c.estimatedDirectness[u1][k1] = 0
				for v1 := k1; v1 < len(c.sortedStations); v1++ {
					nkv := new(big.Float).Set(c.numRoutes[k1][v1])
					nvd := new(big.Float).Set(c.numRoutes[v1][d])
					tuv := new(big.Float).SetFloat64(c.estimatedTransitLength[u1][v1])
					duv := new(big.Float).SetFloat64(Dmatrix[c.sortedStations[u1].S.ID][c.sortedStations[v1].S.ID])

					inc, _ := nkv.Mul(nkv, nvd.Mul(nvd, tuv.Quo(tuv, duv))).Float64()
					c.estimatedDirectness[u1][k1] += inc
				}
			}
			for u := k; u < v; u++ {
				nku := new(big.Float).Set(c.numRoutes[k][u])
				nuv := new(big.Float).Set(c.numRoutes[u][v])
				nvd := new(big.Float).Set(c.numRoutes[v][d])
				tuv := new(big.Float).SetFloat64(c.estimatedTransitLength[u][v])
				duv := new(big.Float).SetFloat64(Dmatrix[c.sortedStations[u].S.ID][c.sortedStations[v].S.ID])
				c.partialResults[k].Add(c.partialResults[k], nku.Mul(nku, nuv.Mul(nuv, nvd.Mul(nvd, tuv.Quo(tuv, duv)))))
			}
		}
	}

	// for i := 0; i < len(sortedStations); i++ {
	// 	for j := i; j < len(sortedStations); j++ {
	// 		log.Printf("%d -> %d: estimated %.05f direct %.05f", sortedStations[i].S.ID, sortedStations[j].S.ID, c.estimatedTransitLength[i][j], Dmatrix[sortedStations[i].S.ID][sortedStations[j].S.ID])
	// 	}
	// }
}

// PredictStationGain predicts the directness
func (c *RouteDirectnessCriterion) PredictStationGain(path []*stationgraph.StationNode, choice *stationgraph.StationNode) float64 {
	k := c.stationIndexMap[choice.S.ID]
	d := len(c.sortedStations) - 1
	result := big.NewFloat(0)
	for _, s := range path {
		u := c.stationIndexMap[s.S.ID]
		result.Add(result, new(big.Float).SetFloat64(c.estimatedDirectness[u][k]))
	}
	result.Add(result, c.partialResults[k])
	nkd := new(big.Float).Set(c.numRoutes[k][d])
	fresult, _ := result.Quo(result, nkd).Float64()
	return fresult
}
