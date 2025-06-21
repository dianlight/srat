package homeassistant

// Docs https://github.com/oapi-codegen/oapi-codegen?tab=readme-ov-file#generating-api-clients

//go:generate go tool oapi-codegen -config cfg.yaml  -package core -o core/client.gen.go core.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package core_api -o core_api/client.gen.go core_api.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package hardware -o hardware/client.gen.go hardware.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package mount -o mount/client.gen.go mount.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package ingress -o ingress/client.gen.go ingress.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package host -o host/client.gen.go host.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package addons -o addons/client.gen.go addons.yaml
//go:generate go tool oapi-codegen -config cfg.yaml  -package resolution -o resolution/client.gen.go resolution.yaml
