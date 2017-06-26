package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ecix/alice-lg/backend/api"

	"github.com/julienschmidt/httprouter"
)

// Alice LG Rest API
//
// The API provides endpoints for getting
// information from the routeservers / alice datasources.
//
// Endpoints:
//
//   Config
//     Show         /api/config
//
//   Routeservers
//     List         /api/routeservers
//     Status       /api/routeservers/:id/status
//     Neighbours   /api/routeservers/:id/neighbours
//     Routes       /api/routeservers/:id/neighbours/:neighbourId/routes
//
//   Querying
//     LookupPrefix /api/routeservers/:id/lookup/prefix?q=<prefix>
//

type apiEndpoint func(*http.Request, httprouter.Params) (api.Response, error)

// Wrap handler for access controll, throtteling and compression
func endpoint(wrapped apiEndpoint) httprouter.Handle {
	return func(res http.ResponseWriter,
		req *http.Request,
		params httprouter.Params) {

		// Get result from handler
		result, err := wrapped(req, params)
		if err != nil {
			result = api.ErrorResponse{
				Error: err.Error(),
			}
			payload, _ := json.Marshal(result)
			http.Error(res, string(payload), http.StatusInternalServerError)
			return
		}

		// Encode json
		payload, err := json.Marshal(result)

		// Set response header
		res.Header().Set("Content-Type", "application/json")

		// Check if compression is supported
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			// Compress response
			res.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(res)
			defer gz.Close()
			gz.Write(payload)
		} else {
			res.Write(payload) // Fall back to uncompressed response
		}
	}
}

// Register api endpoints
func apiRegisterEndpoints(router *httprouter.Router) error {

	// Meta
	router.GET("/api/status", endpoint(apiStatusShow))
	router.GET("/api/config", endpoint(apiConfigShow))

	// Routeservers
	router.GET("/api/routeservers",
		endpoint(apiRouteserversList))
	router.GET("/api/routeservers/:id/status",
		endpoint(apiStatus))
	router.GET("/api/routeservers/:id/neighbours",
		endpoint(apiNeighboursList))
	router.GET("/api/routeservers/:id/neighbours/:neighbourId/routes",
		endpoint(apiRoutesList))

	// Querying
	router.GET("/api/lookup/prefix",
		endpoint(apiLookupPrefixGlobal))

	return nil
}

// Handle Status Endpoint, this is intended for
// monitoring and service health checks
func apiStatusShow(_req *http.Request, _params httprouter.Params) (api.Response, error) {
	status, err := NewAppStatus()
	return status, err
}

// Handle Config Endpoint
func apiConfigShow(_req *http.Request, _params httprouter.Params) (api.Response, error) {
	result := api.ConfigResponse{
		Rejection: api.Rejection{
			Asn:      AliceConfig.Ui.RoutesRejections.Asn,
			RejectId: AliceConfig.Ui.RoutesRejections.RejectId,
		},
		RejectReasons: AliceConfig.Ui.RoutesRejections.Reasons,
		Noexport: api.Noexport{
			Asn:        AliceConfig.Ui.RoutesNoexports.Asn,
			NoexportId: AliceConfig.Ui.RoutesNoexports.NoexportId,
		},
		NoexportReasons: AliceConfig.Ui.RoutesNoexports.Reasons,
		RoutesColumns:   AliceConfig.Ui.RoutesColumns,
	}
	return result, nil
}

// Handle Routeservers List
func apiRouteserversList(_req *http.Request, _params httprouter.Params) (api.Response, error) {
	// Get list of sources from config,
	routeservers := []api.Routeserver{}

	sources := AliceConfig.Sources
	for id, source := range sources {
		routeservers = append(routeservers, api.Routeserver{
			Id:   id,
			Name: source.Name,
		})
	}

	// Make routeservers response
	response := api.RouteserversResponse{
		Routeservers: routeservers,
	}

	return response, nil
}

// Helper: Validate source Id
func validateSourceId(id string) (int, error) {
	rsId, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	if rsId < 0 {
		return 0, fmt.Errorf("Source id may not be negative")
	}
	if rsId >= len(AliceConfig.Sources) {
		return 0, fmt.Errorf("Source id not within [0, %d]", len(AliceConfig.Sources)-1)
	}

	return rsId, nil
}

// Helper: Validate query string
func validateQueryString(req *http.Request, key string) (string, error) {
	query := req.URL.Query()
	values, ok := query[key]
	if !ok {
		return "", fmt.Errorf("Query param %s is missing.", key)
	}

	if len(values) != 1 {
		return "", fmt.Errorf("Query param %s is ambigous.", key)
	}

	value := values[0]
	if value == "" {
		return "", fmt.Errorf("Query param %s may not be empty.", key)
	}

	return value, nil
}

// Helper: Validate prefix query
func validatePrefixQuery(value string) (string, error) {

	// We should at least provide 2 chars
	if len(value) < 2 {
		return "", fmt.Errorf("Query too short")
	}

	// Query constraints: Should at least include a dot or colon
	/* let's try without this :)

	if strings.Index(value, ".") == -1 &&
		strings.Index(value, ":") == -1 {
		return "", fmt.Errorf("Query needs at least a ':' or '.'")
	}
	*/

	return value, nil
}

// Handle status
func apiStatus(_req *http.Request, params httprouter.Params) (api.Response, error) {
	rsId, err := validateSourceId(params.ByName("id"))
	if err != nil {
		return nil, err
	}
	source := AliceConfig.Sources[rsId].getInstance()
	result, err := source.Status()
	return result, err
}

// Handle get neighbours on routeserver
func apiNeighboursList(_req *http.Request, params httprouter.Params) (api.Response, error) {
	rsId, err := validateSourceId(params.ByName("id"))
	if err != nil {
		return nil, err
	}
	source := AliceConfig.Sources[rsId].getInstance()
	result, err := source.Neighbours()
	return result, err
}

// Handle routes
func apiRoutesList(_req *http.Request, params httprouter.Params) (api.Response, error) {
	rsId, err := validateSourceId(params.ByName("id"))
	if err != nil {
		return nil, err
	}
	neighbourId := params.ByName("neighbourId")
	source := AliceConfig.Sources[rsId].getInstance()
	result, err := source.Routes(neighbourId)
	return result, err
}

// Handle global lookup
func apiLookupPrefixGlobal(req *http.Request, params httprouter.Params) (api.Response, error) {
	// Get prefix to query
	prefix, err := validateQueryString(req, "q")
	if err != nil {
		return nil, err
	}

	prefix, err = validatePrefixQuery(prefix)
	if err != nil {
		return nil, err
	}

	// Make response
	t0 := time.Now()
	routes := AliceRoutesStore.Lookup(prefix)

	queryDuration := time.Since(t0)
	response := api.LookupResponseGlobal{
		Routes: routes,
		Time:   float64(queryDuration) / 1000.0 / 1000.0, // nano -> micro -> milli
	}
	return response, nil
}
