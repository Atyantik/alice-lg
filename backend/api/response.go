package api

import (
	"time"
)

// Details, usually the original backend response
type Details map[string]interface{}

// Status
type ApiStatus struct {
	Version         string      `json:"version"`
	CacheStatus     CacheStatus `json:"cache_status"`
	ResultFromCache bool        `json:"result_from_cache"`
}

type CacheStatus struct {
	CachedAt time.Time `json:"cached_at"`
	OrigTtl  int       `json:"orig_ttl"`
}

type Status struct {
	ServerTime   time.Time `json:"server_time"`
	LastReboot   time.Time `json:"last_reboot"`
	LastReconfig time.Time `json:"last_reconfig"`
	Message      string    `json:"message"`
	RouterId     string    `json:"router_id"`
	Version      string    `json:"version"`
	Backend      string    `json:"backend"`
}

type StatusResponse struct {
	Api    ApiStatus `json:"api"`
	Ttl    time.Time `json:"ttl"`
	Status Status    `json:"status"`
}

// Routeservers
type Routeserver struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type RouteserversResponse struct {
	Routeservers []Routeserver `json:"routeservers"`
}

// Neighbours
type Neighbour struct {
	Id string `json:"id"`

	// Mandatory fields
	Address         string `json:"address"`
	Asn             int    `json:"asn"`
	State           string `json:"state"`
	Description     string `json:"description"`
	RoutesReceived  int    `json:"routes_received"`
	RoutesFiltered  int    `json:"routes_filtered"`
	RoutesExported  int    `json:"routes_exported"`
	RoutesPreferred int    `json:"routes_preferred"`
	Uptime          int    `json:"uptime"`

	// Original response
	Details map[string]interface{} `json:"details"`
}

type NeighboursResponse struct {
	Api        ApiStatus   `json:"api"`
	Ttl        time.Time   `json:"ttl"`
	Neighbours []Neighbour `json:"neighbours"`
}

// BGP
type Community []int

type BgpInfo struct {
	AsPath      []int       `json:"as_path"`
	NextHop     string      `json:"next_hop"`
	Communities []Community `json:"communities"`
	LocalPref   string      `json:"local_pref"`
	Med         string      `json:"med"`
}

// Prefixes
type Prefix struct {
	Network   string    `json:"network"`
	Interface string    `json:"interface"`
	Metric    int       `json:"metric"`
	Bgp       BgpInfo   `json:"bgp"`
	Age       time.Time `json:"age"`
	Flags     []string  `json:"flags"` // [BGP, unicast, univ]

	Details Details `json:"details"`
}