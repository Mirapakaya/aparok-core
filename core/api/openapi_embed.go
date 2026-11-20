package api

import _ "embed"

//go:embed docs/v1/openapi.yaml
var openapiV1YAML []byte

//go:embed docs/v1/openapi.json
var openapiV1JSON []byte

//go:embed docs/v2/openapi.yaml
var openapiV2YAML []byte

//go:embed docs/v2/openapi.json
var openapiV2JSON []byte
