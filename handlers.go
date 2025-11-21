package main

import (
	"database/sql"
	"net/http"
)

type Handler struct {
	Config Config
	DB     *sql.DB
}

func NewHandler(config Config, db *sql.DB) *Handler {
	return &Handler{Config: config, DB: db}
}

func (h *Handler) TeamAddHandler(w http.ResponseWriter, r *http.Request) {
}

func (h *Handler) TeamGetHandler(w http.ResponseWriter, r *http.Request) {

}
