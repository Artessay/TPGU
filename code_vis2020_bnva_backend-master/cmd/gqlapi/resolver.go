package gqlapi

import (
	"context"
	"strconv"
	"sync"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/algorithm"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/skyline"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/stationgraph"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct {
	StationGraph   map[string]*stationgraph.StationGraph
	MonteCarloTree map[string]*algorithm.MonteCarloTree
	TreeChannels   map[string]chan *algorithm.MonteCarloTree
	RouteList      map[string]*skyline.RouteList
	usrCount       int // used for user naming
	mu             sync.Mutex
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

func New() Config {
	return Config{
		Resolvers: &Resolver{
			StationGraph:   map[string]*stationgraph.StationGraph{},
			MonteCarloTree: map[string]*algorithm.MonteCarloTree{},
			TreeChannels:   map[string]chan *algorithm.MonteCarloTree{},
			RouteList:      map[string]*skyline.RouteList{},
			usrCount:       0,
		},
	}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateRoutePlanning(ctx context.Context, input NewRoutePlanning) (*skyline.RouteList, error) {
	panic("not implemented")
}

func (r *mutationResolver) CreateStationGraph(ctx context.Context, input NewStationGraph) (*AfterGraphBuilt, error) {
	// panic("not implemented")
	graph, tree, channel, list := LoadGraphFromData(input.Origin, input.Dest, input.Stops)

	r.mu.Lock()

	// raw usr naming
	username := strconv.Itoa(r.usrCount)
	r.usrCount++
	r.MonteCarloTree[username] = tree
	r.RouteList[username] = list
	r.StationGraph[username] = graph
	r.TreeChannels[username] = channel

	r.mu.Unlock()

	rval := AfterGraphBuilt{
		Username: username,
		Graph:    graph,
	}

	return &rval, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Plannings(ctx context.Context, username string) (*skyline.RouteList, error) {
	// panic("not implemented")
	return r.RouteList[username], nil
}

func (r *queryResolver) Stations(ctx context.Context) ([]*station.Station, error) {
	return station.GetAllStationPts(), nil
}

func (r *queryResolver) ExistBusRouteSet(ctx context.Context) ([]*ExistBusRoute, error) {
	// read from
	// db := ConnectToPostgre()
	// defer db.Close()
	// routes := GetAllExisitedBusRoutes(db)

	routes := LoadBusRoutesFromCSVFile("./data/bus_routes.csv")
	return routes, nil
}

func (r *queryResolver) Locations(ctx context.Context, idxs []int) ([]*Location, error) {
	retval := []*Location{}
	pts := [][2]float64{}
	for _, idx := range idxs {
		location := LocationIndex[idx]
		pts = append(pts, [2]float64{location.Lat, location.Lon})
		retval = append(retval, location)
	}

	return GetOSRMRouteResult(pts), nil
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) MonteCarloTreeStateChanged(ctx context.Context, username string) (<-chan *algorithm.MonteCarloTree, error) {
	return r.TreeChannels[username], nil
}
