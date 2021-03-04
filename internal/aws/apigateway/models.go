package apigateway

import (
	"fmt"
	"wekactl/internal/env"
)

type RestApiGateway struct {
	Id     string
	Name   string
	ApiKey string
}

func (r RestApiGateway) Url() string {
	return fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/default/%s", r.Id, env.Config.Region, r.Name)
}
