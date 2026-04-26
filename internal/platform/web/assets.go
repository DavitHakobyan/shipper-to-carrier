package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func NewHandler() (http.Handler, error) {
	subtree, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}

	return http.FileServer(http.FS(subtree)), nil
}
