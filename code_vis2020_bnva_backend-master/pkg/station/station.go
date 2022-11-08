package station

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

var stations []Station
var stationPts []*Station
var stationIndex map[int]int
var stationDistance map[int]map[int]float64

func atoi(s string) int {
	value, _ := strconv.Atoi(s)
	return value
}

func atolf(s string) float64 {
	value, _ := strconv.ParseFloat(s, 64)
	return value
}

// LoadStationsFromFile loads bus stations from a tsv file.
func LoadStationsFromFile(path string) {
	abspath, _ := filepath.Abs(path)
	log.WithField("path", abspath).Info("loading stations...")

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	stationIndex = make(map[int]int)

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		s := Station{
			ID:       atoi(columns[0]),
			Name:     columns[1],
			Lon:      atolf(columns[2]),
			Lat:      atolf(columns[3]),
			NumTrips: atoi(columns[4]),
		}
		stationIndex[s.ID] = len(stations)
		// log.Printf("%d -> %d\n", s.ID, stationIndex[s.ID])
		stations = append(stations, s)
		stationPts = append(stationPts, &s)
	}
}

// LoadStationNeighborsFromFile loads bus stations' neighbors from a tsv file.
func LoadStationNeighborsFromFile(path string) {
	abspath, _ := filepath.Abs(path)
	log.WithField("path", abspath).Info("loading station neighbors...")

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		id, _ := strconv.Atoi(columns[0])
		if columns[1] != "nil" {
			for _, nidStr := range strings.Split(columns[1], ",") {
				stations[stationIndex[id]].Neighbors =
					append(stations[stationIndex[id]].Neighbors, atoi(nidStr))
			}
		}
	}
}

// LoadStationDistanceFromFile loads the distance between stations from a TSV file
func LoadStationDistanceFromFile(path string) {
	abspath, _ := filepath.Abs(path)
	log.WithField("path", abspath).Info("loading station distances...")

	fp, err := os.Open(abspath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	stationDistance = make(map[int]map[int]float64)

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), "\t")
		src, _ := strconv.Atoi(columns[0])
		dst, _ := strconv.Atoi(columns[1])
		dist, _ := strconv.ParseFloat(columns[2], 64)
		if stationDistance[src] == nil {
			stationDistance[src] = make(map[int]float64)
		}
		stationDistance[src][dst] = dist
	}
}

// GetAllStations returns all stations in a Slice.
func GetAllStations() []Station {
	return stations
}

// GetAllStationPts return all pointers of stations in a Slice
func GetAllStationPts() []*Station {
	return stationPts
}

// GetStationByID returns the station with the ID
func GetStationByID(id int) *Station {
	return &stations[stationIndex[id]]
}

// GetStationDistanceByIDs returns the distance between two stations
func GetStationDistanceByIDs(srcID, dstID int) float64 {
	return stationDistance[srcID][dstID]
}

// GetDistanceMatrix returns the distance matrix
func GetDistanceMatrix() map[int]map[int]float64 {
	return stationDistance
}
