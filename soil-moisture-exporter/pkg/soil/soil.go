package soil

type Reading struct {
	SensorName string `json:"name"`
	Value float64 `json:"value"`
}