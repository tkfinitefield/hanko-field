package public

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var static embed.FS

func StaticFS() (fs.FS, error) {
	return fs.Sub(static, "static")
}

