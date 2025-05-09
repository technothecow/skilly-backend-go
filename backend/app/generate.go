package codegen

//go:generate oapi-codegen -config api/codegen.cfg.yaml -package gen -generate types,gin-server,embedded-spec,models api/openapi.yaml
