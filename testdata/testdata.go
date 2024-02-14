package testdata

import (
	"embed"
	_ "embed"
)

//go:embed private.pem
var Private []byte

//go:embed public.pem
var Public []byte

//go:embed *.pem
var FS embed.FS
