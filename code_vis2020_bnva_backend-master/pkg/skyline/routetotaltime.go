package skyline

import (
	"log"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// RouteTotalTimeCriterion defines the total time criterion for the routes
type RouteTotalTimeCriterion struct {
	sortedStations   []*stationgraph.StationNode
	stationIndexMap  map[int]int
	averageNumStops  []float64
	averageTotalTime []float64
	count            []float64
}

// Name returns the name of criterion
func (c *RouteTotalTimeCriterion) Name() string {
	return "T"
}

// Evaluate returns the computed criterion value for a route
func (c *RouteTotalTimeCriterion) Evaluate(r []*station.Station) float64 {
	var result float64
	if len(r) == 1 {
		log.Printf("warn: route only has one station %d\n", r[0].ID)
	}
	for i := range r[1:] {
		result += Tmatrix[r[i].ID][r[i+1].ID]
	}
	return result + float64((len(r)-2)*90)
}

// GreaterThan returns whether the criterion a is larger than b
func (c *RouteTotalTimeCriterion) GreaterThan(a, b float64) bool {
	return a <= b
}

// Normalize returns the normalized criterion value between 0 and 1
func (c *RouteTotalTimeCriterion) Normalize(v, min, max float64) float64 {
	return (max-v)/(max-min)*0.95 + 0.05
}

// InitEstimator initializes the criterion
func (c *RouteTotalTimeCriterion) InitEstimator(sortedStations []*stationgraph.StationNode, stationIndexMap map[int]int) {
	c.sortedStations = sortedStations
	c.stationIndexMap = stationIndexMap
	c.averageNumStops = make([]float64, len(c.sortedStations))
	c.averageTotalTime = make([]float64, len(c.sortedStations))
	c.count = make([]float64, len(c.sortedStations))
	c.count[len(c.sortedStations)-1] = 1

	o := 0
	d := len(c.sortedStations) - 1
	for i := d - 1; i >= o; i-- {
		for _, ns := range c.sortedStations[i].Next {
			nsp := c.stationIndexMap[ns.S.ID]
			c.count[i] += c.count[nsp]
			// c.averageNumStops[i] += c.averageNumStops[nsp] + 1
			c.averageTotalTime[i] += c.count[nsp] * (c.averageTotalTime[nsp] + Tmatrix[c.sortedStations[i].S.ID][ns.S.ID] + 90)
		}
		// c.averageNumStops[i] /= float64(len(c.sortedStations[i].Next))
		c.averageTotalTime[i] /= c.count[i]
	}
}

// PredictStationGain returns the gain for choosing a station while a route prefix is known
func (c *RouteTotalTimeCriterion) PredictStationGain(path []*stationgraph.StationNode, choice *stationgraph.StationNode) float64 {
	s := path[len(path)-1]
	k := c.stationIndexMap[choice.S.ID]
	return Tmatrix[s.S.ID][choice.S.ID] + 90 + c.averageTotalTime[k]
}

// EstimateStationGain sets the gain for the specific criteria at a station
// func (c *RouteTotalTimeCriterion) EstimateStationGain(
// 	s *stationgraph.StationNode,
// 	sortedStations []*stationgraph.StationNode,
// 	stationIndexMap map[int]int,
// ) float64 {
// 	i := stationIndexMap[s.S.ID]

// 	visited := make([]bool, len(sortedStations))
// 	depth := make([]float64, len(sortedStations))
// 	time := make([]float64, len(sortedStations))

// 	visited[i] = true
// 	depth[i] = 1.0
// 	time[i] = 0.0

// 	for j := i - 1; j >= 0; j-- {
// 		var (
// 			d, t float64
// 			cnt  int
// 		)
// 		for _, ns := range sortedStations[j].Next {
// 			nsp := stationIndexMap[ns.S.ID]
// 			if nsp <= i && visited[nsp] {
// 				visited[j] = true
// 				d += depth[nsp]
// 				t += depth[nsp] + Tmatrix[sortedStations[j].S.ID][ns.S.ID]
// 				cnt++
// 			}
// 		}
// 		depth[j] = d / float64(cnt)
// 		time[j] = t / float64(cnt)
// 	}
// 	// for j := i + 1; j < len(sortedStations); j++ {
// 	// 	var (
// 	// 		d, t float64
// 	// 		cnt  int
// 	// 	)
// 	// 	for _, ps := range sortedStations[j].Prev {
// 	// 		psp := stationIndexMap[ps.S.ID]
// 	// 		if psp >= i && visited[psp] {
// 	// 			visited[j] = true
// 	// 			d += depth[psp]
// 	// 			t += depth[psp] + Tmatrix[ps.S.ID][sortedStations[j].S.ID]
// 	// 			cnt++
// 	// 		}
// 	// 	}
// 	// 	depth[j] = d / float64(cnt)
// 	// 	time[j] = t / float64(cnt)
// 	// }

// 	o := 0
// 	d := len(sortedStations) - 1

// 	return (depth[o]+depth[d]-3)*90 + time[o] + time[d]
// }
