package station

// Station declares the bus station type.
type Station struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Lon       float64 `json:"lon"`
	Lat       float64 `json:"lat"`
	NumTrips  int     `json:"num_trips"`
	Neighbors []int   `json:"-"`
	InGraph   bool
}

// X returns x coordinate
func (s *Station) X() float64 {
	return s.Lon
}

// Y returns y coordinate
func (s *Station) Y() float64 {
	return s.Lat
}

// Subtract performs vector subtraction between the locations of two stations
func (s *Station) Subtract(t *Station) (float64, float64) {
	return s.X() - t.X(), s.Y() - t.Y()
}
