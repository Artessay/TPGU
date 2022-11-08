package skyline

import (
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// IRouteCriterion defines a criterion for the routes
type IRouteCriterion interface {
	Name() string
	Evaluate(r []*station.Station) float64
	GreaterThan(a, b float64) bool
	Normalize(v, min, max float64) float64

	InitEstimator(sortedStations []*stationgraph.StationNode, stationIndexMap map[int]int)
	PredictStationGain(path []*stationgraph.StationNode, choice *stationgraph.StationNode) float64
	// EstimateStationGain(s *stationgraph.StationNode, sortedStations []*stationgraph.StationNode, stationIndexMap map[int]int) float64
}

// RouteCriteria defines the criteria used in skyline computation
var RouteCriteria = []IRouteCriterion{
	&RouteTotalTimeCriterion{},
	&RouteFlowCriterion{},
	&RouteDirectnessCriterion{},
}

// InitializeEstimators initializes the estimators defined in criteria
func InitializeEstimators(g *stationgraph.StationGraph) {
	sorted := g.SortTopo()

	stationIndexMap := make(map[int]int)
	for i, s := range sorted {
		stationIndexMap[s.S.ID] = i
	}

	// min := make([]float64, len(RC))
	// max := make([]float64, len(RC))
	for _, c := range RouteCriteria {
		c.InitEstimator(sorted, stationIndexMap)
		// min[k] = math.Inf(1)
		// max[k] = math.Inf(-1)
	}

	// for _, s := range sorted {
	// 	for k, c := range RC {
	// 		g := c.EstimateStationGain(s, sorted, stationIndexMap)
	// 		s.EstimatedGain = append(s.EstimatedGain, g)
	// 		if min[k] > g {
	// 			min[k] = g
	// 		}
	// 		if max[k] < g {
	// 			max[k] = g
	// 		}
	// 	}
	// }

	// for i := len(sorted) - 1; i >= 0; i-- {
	// 	for k, c := range RC {
	// 		sorted[i].EstimatedGain[k] = c.Normalize(sorted[i].EstimatedGain[k], min[k], max[k])
	// 	}
	// }
}
