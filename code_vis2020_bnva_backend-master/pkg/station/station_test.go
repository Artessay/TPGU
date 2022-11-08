package station_test

import (
	"testing"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
)

func TestStationIndex(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	for _, s := range station.GetAllStations() {
		u := station.GetStationByID(s.ID)
		if u.ID != s.ID {
			t.Fatalf("ID mismatch: should be %d instead of %d", s.ID, u.ID)
		}
	}
}
