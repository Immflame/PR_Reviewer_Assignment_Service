package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"pr_reviewer_assignment_service/config"
	"pr_reviewer_assignment_service/models"
	"time"
)

type Handler struct {
	Conf config.Config
	DB   *sql.DB
}

func NewHandler(conf config.Config, db *sql.DB) *Handler {
	return &Handler{Conf: conf, DB: db}
}

func (h *Handler) TeamAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.Team

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var team_name string
	query := "INSERT INTO teams (team_name) VALUES ('$1') RETURNING team_name"

	if err := h.DB.QueryRowContext(ctx, query, req.TeamName).Scan(&team_name); err != nil {
		http.Error(w, "Failed to add team", http.StatusInternalServerError)
		return
	}

	for i := 0; i < len(req.Members); i++ {
		query := "INSERT INTO users (user_id, username, team_name, is_active) VALUES ('$1', '$2','$3', '$4')"
		err := h.DB.QueryRowContext(ctx, query, req.Members[i].UserID, req.Members[i].Username, team_name, req.Members[i].IsActive)
		if err != nil {
			http.Error(w, "Failed to add user", http.StatusInternalServerError)
			return
		}
	}

	team := models.Team{
		TeamName: team_name,
		Members:  req.Members,
	}

	resp := models.TeamAddResponse{Team: team}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) TeamGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if teamName := r.URL.Query().Get("name"); teamName == "" {
		http.Error(w, "'name' parameter is required", http.StatusBadRequest)
		return
	}

	query := "SELECT * FROM users WHERE team_name = $1"
	var 

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) UserSetIsActiveHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	// defer cancel()

	// var req models.UserSetIsActiveRequest

	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }

	// ////

	// resp := models.UserSetIsActiveResponse{User: user}
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }

}

func (h *Handler) UserGetReviewHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	// defer cancel()
	// if user_id := r.URL.Query().Get("user_id"); user_id == "" {
	// 	http.Error(w, "'name' parameter is required", http.StatusBadRequest)
	// 	return
	// }

	// ////

	// resp := models.UserGetReviewResponse{UserID: user_id, PullRequests: pullRequests}
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }
}

func (h *Handler) PRCreateHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	// defer cancel()

	// var req models.PRCreateRequest

	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }

	// ////

	// resp := models.PRResponse{PullRequest: pr}
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }
}

func (h *Handler) PRMergeHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	// defer cancel()

	// var req models.PRMergeRequest

	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }

	// ////

	// resp := models.PRResponse{PullRequest: pr}
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }
}

func (h *Handler) PRReassignHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	// defer cancel()

	// var req models.PRReassignRequest

	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }

	// ////

	// resp := models.PRReassignResponse{PullRequest: pr, Replaced_By: replaced_by}
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(resp); err != nil {
	// 	http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	// 	return
	// }
}
