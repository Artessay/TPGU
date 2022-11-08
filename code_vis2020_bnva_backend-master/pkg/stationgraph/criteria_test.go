package stationgraph_test

import (
	"testing"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

func TestIsMovingForward(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	origin := station.GetStationByID(849)
	t.Logf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	dest := station.GetStationByID(12067)
	t.Logf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	curr := station.GetStationByID(849)
	t.Logf("use %d (%f, %f) as the current station", curr.ID, curr.Lon, curr.Lat)
	target := station.GetStationByID(12067)
	t.Logf("use %d (%f, %f) as the target station", target.ID, target.Lon, target.Lat)

	if !stationgraph.IsMovingForward(target, curr, origin, dest) {
		t.Fatal("Test #1: IsMovingForward returns false, should be true")
	} else if stationgraph.IsMovingForward(curr, target, origin, dest) {
		t.Fatal("Test #2: IsMovingForward returns true, should be false")
	} else if !stationgraph.IsMovingForward(curr, target, dest, origin) {
		t.Fatal("Test #3: IsMovingForward returns false, should be true")
	} else if stationgraph.IsMovingForward(target, curr, dest, origin) {
		t.Fatal("Test #4: IsMovingForward returns true, should be false")
	}
}

func TestIsOriginFarther(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")
	origin := station.GetStationByID(849)
	t.Logf("use %d (%f, %f) as the origin station", origin.ID, origin.Lon, origin.Lat)
	curr := station.GetStationByID(849)
	t.Logf("use %d (%f, %f) as the current station", curr.ID, curr.Lon, curr.Lat)
	target := station.GetStationByID(12067)
	t.Logf("use %d (%f, %f) as the target station", target.ID, target.Lon, target.Lat)

	if !stationgraph.IsOriginFarther(target, curr, origin) {
		t.Fatal("Test #1: IsOriginFarther returns false, should be true")
	} else if stationgraph.IsOriginFarther(curr, target, origin) {
		t.Fatal("Test #2: IsOriginFarther returns true, should be false")
	}
}

func TestIsDestinationCloser(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")
	dest := station.GetStationByID(12067)
	t.Logf("use %d (%f, %f) as the destination station", dest.ID, dest.Lon, dest.Lat)
	curr := station.GetStationByID(849)
	t.Logf("use %d (%f, %f) as the current station", curr.ID, curr.Lon, curr.Lat)
	target := station.GetStationByID(12067)
	t.Logf("use %d (%f, %f) as the target station", target.ID, target.Lon, target.Lat)

	if !stationgraph.IsDestinationCloser(target, curr, dest) {
		t.Fatal("Test #1: IsDestinationCloser returns false, should be true")
	} else if stationgraph.IsDestinationCloser(curr, target, dest) {
		t.Fatal("Test #2: IsDestinationCloser returns true, should be false")
	}
}

func TestIsZigzagRoute(t *testing.T) {
	station.LoadStationsFromFile("../../data/bus_stations.tsv")
	station.LoadStationDistanceFromFile("../../data/dist_matrix.tsv")

	route1 := []*station.Station{
		station.GetStationByID(1579),
		station.GetStationByID(4011),
		station.GetStationByID(3385),
	}
	if !stationgraph.IsZigzagRoute(station.GetStationByID(2181), route1) {
		t.Fatal("Test #1: TestIsZigzagRoute returns false, should be true")
	}

	route2 := []*station.Station{
		station.GetStationByID(1579),
		station.GetStationByID(4011),
		station.GetStationByID(2181),
	}
	if stationgraph.IsZigzagRoute(station.GetStationByID(5664), route2) {
		t.Fatal("Test #1: TestIsZigzagRoute returns true, should be false")
	}
}
