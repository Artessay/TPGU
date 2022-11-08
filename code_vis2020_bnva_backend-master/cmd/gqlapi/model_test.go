package gqlapi_test

import (
	"testing"

	"git.zjuvis.org/rnvis/bus-routing-backend/cmd/gqlapi"
)

func TestLoadBusRoutes(t *testing.T) {
	gqlapi.LoadBusRoutes("../../data/bus_routes.csv")
}
