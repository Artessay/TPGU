package stationgraph

import (
	"fmt"
	"math"

	"git.zjuvis.org/rnvis/bus-routing-backend/pkg/station"
)

// IsMovingForward tests the satisfiability of the second criterion
func IsMovingForward(target, curr, origin, dest *station.Station) bool {
	verbose := false

	xn, yn := dest.Subtract(origin)
	theta := math.Atan2(yn, xn)

	xi, yi := curr.Subtract(origin)
	xnewi := xi*math.Cos(theta) + yi*math.Sin(theta)
	txi, tyi := target.Subtract(origin)
	txnewi := txi*math.Cos(theta) + tyi*math.Sin(theta)

	if verbose {
		fmt.Printf("theta: %f\n", theta)
		fmt.Printf("xi, yi: %f, %f\n", xi, yi)
		fmt.Printf("txi, tyi: %f, %f\n", txi, tyi)
		fmt.Printf("txnewi, xnewi: %f, %f\n", txnewi, xnewi)
	}
	return txnewi > xnewi
}

// IsOriginFarther tests the satisfiability of the third criterion
func IsOriginFarther(target, curr, origin *station.Station) bool {
	return station.GetStationDistanceByIDs(origin.ID, target.ID) > station.GetStationDistanceByIDs(origin.ID, curr.ID)
}

// IsDestinationCloser tests the satisfiability of the third criterion
func IsDestinationCloser(target, curr, dest *station.Station) bool {
	return station.GetStationDistanceByIDs(target.ID, dest.ID) < station.GetStationDistanceByIDs(curr.ID, dest.ID)
}
