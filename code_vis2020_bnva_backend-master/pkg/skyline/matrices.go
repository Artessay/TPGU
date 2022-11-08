package skyline

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Fmatrix defines the number of trips from one station to another
var Fmatrix map[int]map[int]int

// Dmatrix defines the distance between a pair of stations
var Dmatrix map[int]map[int]float64

// Tmatrix defines the travel time from one station to another
var Tmatrix map[int]map[int]float64

// BusSpeed defines the average speed of buses in meters per second
const BusSpeed float64 = 50.0 * 1000 / 3600

// LoadFlowMatrixFromFile loads the flow matrix from a TSV file
func LoadFlowMatrixFromFile(path string) {
	abspath, _ := filepath.Abs(path)
	log.WithField("path", abspath).Info("loading flow matrix...")

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	Fmatrix = make(map[int]map[int]int)

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		src, _ := strconv.Atoi(columns[0])
		dst, _ := strconv.Atoi(columns[1])
		count, _ := strconv.Atoi(columns[2])

		if Fmatrix[src] == nil {
			Fmatrix[src] = make(map[int]int)
		}
		Fmatrix[src][dst] = count
	}
}

// ComputeTimeMatrixFromDistance computes the time matrix based on distance
func ComputeTimeMatrixFromDistance(dist map[int]map[int]float64) {
	Tmatrix = make(map[int]map[int]float64)
	Dmatrix = make(map[int]map[int]float64)
	for s, p := range dist {
		Tmatrix[s] = make(map[int]float64)
		Dmatrix[s] = make(map[int]float64)
		for t, d := range p {
			Tmatrix[s][t] = d / BusSpeed
			Dmatrix[s][t] = d
		}
	}
}
