package sources

import (
	"github.com/ecix/alice-lg/backend/api"
)

type Source interface {
	Status() (api.StatusResponse, error)
	Neighbours() (api.NeighboursResponse, error)
	Routes(neighbourId string) (api.RoutesResponse, error)
	AllRoutes() (api.RoutesResponse, error)
}
