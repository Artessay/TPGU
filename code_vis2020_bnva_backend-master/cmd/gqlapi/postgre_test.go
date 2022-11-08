package gqlapi_test

import (
	"log"
	"testing"

	"git.zjuvis.org/rnvis/bus-routing-backend/cmd/gqlapi"
)

func TestConnectToPostgre(t *testing.T) {
	db := gqlapi.ConnectToPostgre()
	routes := gqlapi.GetAllExisitedBusRoutes(db)
	log.Printf("Get %d routes", len(routes))
	db.Close()
}
