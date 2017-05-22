package birdwatcher

// Parsers and helpers

import (
	"sort"
	"strconv"
	"time"

	"github.com/ecix/alice-lg/backend/api"
)

const SERVER_TIME = time.RFC3339Nano
const SERVER_TIME_SHORT = "2006-01-02 15:04:05"
const SERVER_TIME_EXT = "Mon, 2 Jan 2006 15:04:05 +0000"

// Convert server time string to time
func parseServerTime(value interface{}, layout, timezone string) (time.Time, error) {
	svalue, ok := value.(string)
	if !ok {
		return time.Time{}, nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.ParseInLocation(layout, svalue, loc)
	return t, err
}

// Make api status from response:
// The api status is always included in a birdwatcher response
func parseApiStatus(bird ClientResponse, config Config) (api.ApiStatus, error) {
	birdApi := bird["api"].(map[string]interface{})

	ttl, err := parseServerTime(
		bird["ttl"],
		SERVER_TIME,
		config.Timezone,
	)
	if err != nil {
		return api.ApiStatus{}, err
	}

	status := api.ApiStatus{
		Version:         birdApi["Version"].(string),
		ResultFromCache: birdApi["result_from_cache"].(bool),
		Ttl:             ttl,
	}

	return status, nil
}

// Parse neighbour uptime
func parseRelativeServerTime(uptime interface{}, config Config) time.Duration {
	serverTime, _ := parseServerTime(uptime, SERVER_TIME_SHORT, config.Timezone)
	return time.Since(serverTime)
}

// Parse neighbours response
func parseNeighbours(bird ClientResponse, config Config) ([]api.Neighbour, error) {
	neighbours := api.Neighbours{}
	protocols := bird["protocols"].(map[string]interface{})

	// Iterate over protocols map:
	for protocolId, proto := range protocols {
		protocol := proto.(map[string]interface{})
		routes := protocol["routes"].(map[string]interface{})

		uptime := parseRelativeServerTime(protocol["state_changed"], config)
		lastError := mustString(protocol["last_error"], "")

		neighbour := api.Neighbour{
			Id: protocolId,

			Address:     protocol["neighbor_address"].(string),
			Asn:         int(protocol["neighbor_as"].(float64)),
			State:       protocol["state"].(string),
			Description: protocol["description"].(string),

			RoutesReceived:  int(routes["imported"].(float64)),
			RoutesExported:  int(routes["exported"].(float64)),
			RoutesFiltered:  int(routes["filtered"].(float64)),
			RoutesPreferred: int(routes["preferred"].(float64)),

			Uptime:    uptime,
			LastError: lastError,

			Details: protocol,
		}

		neighbours = append(neighbours, neighbour)
	}

	sort.Sort(neighbours)

	return neighbours, nil
}

// Parse route bgp info
func parseRouteBgpInfo(data interface{}) api.BgpInfo {
	bgpData := data.(map[string]interface{})

	asPath := parseIntList(bgpData["as_path"])
	communities := parseBgpCommunities(bgpData["communities"])
	largeCommunities := parseBgpCommunities(bgpData["large_communities"])

	localPref, _ := strconv.Atoi(bgpData["local_pref"].(string))
	medInfo, ok := bgpData["med"].(string)
	med := 0
	if ok {
		med, _ = strconv.Atoi(medInfo)
	}

	bgp := api.BgpInfo{
		Origin:           mustString(bgpData["origin"], "unknown"),
		AsPath:           asPath,
		NextHop:          mustString(bgpData["next_hop"], "unknown"),
		LocalPref:        localPref,
		Med:              med,
		Communities:      communities,
		LargeCommunities: largeCommunities,
	}
	return bgp
}

// Extract bgp communities from response
func parseBgpCommunities(data interface{}) []api.Community {
	communities := []api.Community{}

	ldata, ok := data.([]interface{})
	if !ok { // We don't have any
		return []api.Community{}
	}

	for _, c := range ldata {
		cdata := c.([]interface{})
		community := api.Community{}
		for _, cinfo := range cdata {
			community = append(community, int(cinfo.(float64)))
		}
		communities = append(communities, community)
	}

	return communities
}

// Assert string, provide default
func mustString(value interface{}, fallback string) string {
	sval, ok := value.(string)
	if !ok {
		return fallback
	}
	return sval
}

// Assert list of strings
func mustStringList(data interface{}) []string {
	list := []string{}
	ldata := data.([]interface{})
	for _, e := range ldata {
		s, ok := e.(string)
		if ok {
			list = append(list, s)
		}
	}
	return list
}

// Convert list of strings to int
func parseIntList(data interface{}) []int {
	list := []int{}
	sdata := mustStringList(data)
	for _, e := range sdata {
		val, _ := strconv.Atoi(e)
		list = append(list, val)
	}
	return list
}

// Parse routes response
func parseRoutes(bird ClientResponse, config Config) ([]api.Route, error) {
	routes := api.Routes{}
	birdRoutes := bird["routes"].([]interface{})

	for _, data := range birdRoutes {
		rdata := data.(map[string]interface{})

		age := parseRelativeServerTime(rdata["age"], config)
		rtype := mustStringList(rdata["type"])
		bgpInfo := parseRouteBgpInfo(rdata["bgp"])

		route := api.Route{
			Id:          rdata["network"].(string),
			NeighbourId: rdata["from_protocol"].(string),

			Network:   rdata["network"].(string),
			Interface: rdata["interface"].(string),
			Gateway:   rdata["gateway"].(string),
			Metric:    int(rdata["metric"].(float64)),
			Age:       age,
			Type:      rtype,
			Bgp:       bgpInfo,

			Details: rdata,
		}

		routes = append(routes, route)
	}

	// Sort routes
	sort.Sort(routes)

	return routes, nil
}
