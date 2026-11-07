package api

import _ "embed"

//go:embed openapi/v1/openapi.yaml
var OpenAPISpec []byte
