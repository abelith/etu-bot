package models

type Inhabitant struct {
	Id           int
	Name         string
	HouseAddress string
}

type House struct {
	Address     string
	Latitude    float64
	Longitude   float64
	Inhabitants int
}

type Cluster struct {
	Name   string
	Houses []*House
}
