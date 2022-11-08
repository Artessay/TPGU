package skyline

import (
	"log"
	"math"
	"sort"
	"sync"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
)

// Route describes a route that is not dominated by others
type Route struct {
	R          []*station.Station `json:"R"`
	Criteria   []float64          `json:"C"`
	Hash       int                `json:"-"`
	RevokeFunc func()             `json:"-"`
}

// Dominate tests if sr dominates nsr
func (sr *Route) Dominate(nsr *Route, cds []IRouteCriterion) bool {
	for i, cd := range cds {
		if !cd.GreaterThan(sr.Criteria[i], nsr.Criteria[i]) {
			return false
		}
	}
	return true
}

// Print prints a skyline route
func (sr *Route) Print(cds []IRouteCriterion) {
	var IDs []int
	for _, s := range sr.R {
		IDs = append(IDs, s.ID)
	}
	log.Printf("Route %v", IDs)
	for i, cd := range cds {
		log.Printf("  %s=%.5f\n", cd.Name(), sr.Criteria[i])
	}
}

// RouteList describes a list of skyline routes
type RouteList struct {
	Mutex               sync.Mutex
	Routes              []*Route
	CriteriaDescriptors []IRouteCriterion
	NumAdds             int
	NumDeletion			int
}

func getRouteHash(nr []*station.Station) int {
	hash := 0
	for i := range nr {
		hash = hash*17 + nr[i].ID + nr[len(nr)-i-1].ID
		hash %= 121527269
	}
	return hash
}

// GetRouteByHash returns route by their hash
func (list *RouteList) GetRouteByHash(nr []*station.Station, hash int) *Route {
	for _, r := range list.Routes {
		if r.Hash == hash && len(r.R) == len(nr) {
			fwdsame, bkwdsame := true, true
			for i := 0; i < len(nr) && (fwdsame || bkwdsame); i++ {
				if nr[i].ID != r.R[i].ID {
					fwdsame = false
				}
				if nr[i].ID != r.R[len(nr)-i-1].ID {
					bkwdsame = false
				}
			}
			if fwdsame || bkwdsame {
				return r
			}
		}
	}
	return nil
}

// Add adds a route into a skyline route list
func (list *RouteList) Add(nr []*station.Station, revokeFunc func()) (bool, []float64) {
	list.Mutex.Lock()
	defer list.Mutex.Unlock()

	list.NumAdds++

	hash := getRouteHash(nr)
	if dr := list.GetRouteByHash(nr, hash); dr != nil {
		return false, dr.Criteria
	}

	var values []float64
	for _, cd := range list.CriteriaDescriptors {
		values = append(values, cd.Evaluate(nr))
	}

	nsr := &Route{
		R:          nr,
		Hash:       hash,
		Criteria:   values,
		RevokeFunc: revokeFunc,
	}

	// Build new Routes set based on domination relationship
	isSkyline := true
	var newRoutes []*Route
	for _, sr := range list.Routes {
		if isSkyline {
			if sr.Dominate(nsr, list.CriteriaDescriptors) {
				isSkyline = false
				newRoutes = append(newRoutes, sr)
			} else if nsr.Dominate(sr, list.CriteriaDescriptors) {
				// find an existed route being dominate -> delete the route 
				list.NumDeletion++;
				if sr.RevokeFunc != nil {
					sr.RevokeFunc()
				}
			} else {
				newRoutes = append(newRoutes, sr)
			}
		} else {
			newRoutes = append(newRoutes, sr)
		}
	}
	if isSkyline {
		newRoutes = append(newRoutes, nsr)
	}
	list.Routes = newRoutes

	return isSkyline, nsr.Criteria
}

// Print prints a skyline route list
func (list *RouteList) Print() {
	sort.Slice(list.Routes, func(i, j int) bool {
		for k := 0; k < len(list.Routes[i].Criteria); k++ {
			if math.Abs(list.Routes[i].Criteria[k]-list.Routes[j].Criteria[k]) > 1e-7 {
				return list.Routes[i].Criteria[k] > list.Routes[j].Criteria[k]
			}
		}
		return false
	})
	for _, r := range list.Routes {
		r.Print(list.CriteriaDescriptors)
	}
	log.Printf("Number of additions: %d", list.NumAdds)
}

// BuildRouteList builds a RouteList
func BuildRouteList() *RouteList {
	return &RouteList{
		CriteriaDescriptors: RouteCriteria,
	}
}
