package api

import (
	"database/sql"
	"net/http"
)

func NewServeMux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	registerRoutes(mux, db)
	return mux
}
