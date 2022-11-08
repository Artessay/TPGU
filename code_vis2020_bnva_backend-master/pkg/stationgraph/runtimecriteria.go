package stationgraph

import "git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"

// IsZigzagRoute tests whether a route is zigzag
func IsZigzagRoute(target *station.Station, route []*station.Station) bool {
	if len(route) == 0 {
		return false
	}

	minDist := station.GetStationDistanceByIDs(route[len(route)-1].ID, target.ID)
	for k := len(route) - 2; k >= 0; k-- {
		if station.GetStationDistanceByIDs(route[k].ID, target.ID) < minDist {
			return true
		}
	}

	return false
}
