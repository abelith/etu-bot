package models

type Priority = uint32

const (
	Lowest Priority = iota
	Low
	LowerMedium
	Medium
	High
	Highest
)

type User struct {
	Id     int
	Name   string
	Active bool
}

type Inhabitant struct {
	User         User
	HouseAddress string
}

type OrgMember struct {
	User  User
	OrgID int
	Role  string
}

type Organization struct {
	Id          int
	Name        string
	Description string
}

type ClusterStat struct {
	Name         string
	HousesNumber int
}

type OrgMemberMe struct {
	OrgMember   OrgMember
	Org         Organization
	ClusterStat ClusterStat
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

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Notification struct {
	Id          int
	SourceID    string
	ClusterName string
	Priority    Priority
	Tags        []string
}
