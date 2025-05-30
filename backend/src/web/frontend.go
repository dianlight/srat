//go:build embedallowed

package web

import "embed"

//go:generate make -C ../.. static

//go:embed static/*
var Static_content embed.FS
