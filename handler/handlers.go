package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"pr_reviewer_assignment_service/config"
	"pr_reviewer_assignment_service/models"
	"time"

	"math/rand"
)

type Handler struct {
	Conf config.Config
	DB   *sql.DB
}

func NewHandler(conf config.Config, db *sql.DB) *Handler {
	return &Handler{Conf: conf, DB: db}
}

func sendErrorResponse(w http.ResponseWriter, httpStatus int, errorCode models.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	resp := models.ErrorResponse{
		Error: models.ErrorDetails{
			Code:    errorCode,
			Message: message,
		},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

func pickNRandomStrings(candidates []string, n int) []string {
	if n <= 0 || len(candidates) == 0 {
		return []string{}
	}

	if n >= len(candidates) {
		return candidates
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shuffled := make([]string, len(candidates))
	copy(shuffled, candidates)
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:n]
}

func (h *Handler) TeamAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var existingTeamName string
	err = tx.QueryRowContext(ctx, "SELECT team_name FROM teams WHERE team_name = $1", req.TeamName).Scan(&existingTeamName)
	if err == nil {
		sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeTeamExists, "Team with this name already exists")
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("Failed to check existing team: %v", err)
		http.Error(w, "Failed to check existing team", http.StatusInternalServerError)
		return
	}

	insertTeamQuery := "INSERT INTO teams (team_name) VALUES ($1) RETURNING team_name"
	var teamName string
	if err := tx.QueryRowContext(ctx, insertTeamQuery, req.TeamName).Scan(&teamName); err != nil {
		log.Printf("Failed to add team: %v", err)
		http.Error(w, "Failed to add team", http.StatusInternalServerError)
		return
	}

	var addedMembers []models.TeamMember
	for _, member := range req.Members {
		upsertUserQuery := `
			INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT (user_id) DO UPDATE SET
				username = EXCLUDED.username,
				team_name = EXCLUDED.team_name,
				is_active = EXCLUDED.is_active,
				updated_at = CURRENT_TIMESTAMP
			RETURNING user_id, username, is_active;
		`
		var updatedUserID, updatedUsername string
		var updatedIsActive bool
		err := tx.QueryRowContext(ctx, upsertUserQuery, member.UserID, member.Username, teamName, member.IsActive).Scan(&updatedUserID, &updatedUsername, &updatedIsActive)
		if err != nil {
			log.Printf("Failed to upsert user %s: %v", member.UserID, err)
			http.Error(w, "Failed to add/update user", http.StatusInternalServerError)
			return
		}
		addedMembers = append(addedMembers, models.TeamMember{
			UserID:   updatedUserID,
			Username: updatedUsername,
			IsActive: updatedIsActive,
		})
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	team := models.Team{
		TeamName: teamName,
		Members:  addedMembers,
	}

	resp := models.TeamAddResponse{Team: team}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) TeamGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "'team_name' parameter is required", http.StatusBadRequest)
		return
	}

	var existingTeamName string
	err := h.DB.QueryRowContext(ctx, "SELECT team_name FROM teams WHERE team_name = $1", teamName).Scan(&existingTeamName)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Team not found")
		return
	}
	if err != nil {
		log.Printf("Failed to query team: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query := "SELECT user_id, username, is_active FROM users WHERE team_name = $1"
	rows, err := h.DB.QueryContext(ctx, query, teamName)
	if err != nil {
		log.Printf("Failed to query team members: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			log.Printf("Failed to scan team member: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows for team members: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	team := models.Team{
		TeamName: teamName,
		Members:  members,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(team); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) UserSetIsActiveHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.UserSetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE users
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2
		RETURNING user_id, username, team_name, is_active;
	`
	var user models.User
	err := h.DB.QueryRowContext(ctx, query, req.IsActive, req.UserID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
	)

	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "User not found")
		return
	}
	if err != nil {
		log.Printf("Failed to update user activity status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := models.UserSetIsActiveResponse{User: user}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) UserGetReviewHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "'user_id' parameter is required", http.StatusBadRequest)
		return
	}

	var existingUserID string
	err := h.DB.QueryRowContext(ctx, "SELECT user_id FROM users WHERE user_id = $1", userID).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "User not found")
		return
	}
	if err != nil {
		log.Printf("Failed to query user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query := `
		SELECT pull_request_id, pull_request_name, author_id, status
		FROM pull_requests
		WHERE reviewer1_id = $1 OR reviewer2_id = $1;
	`
	rows, err := h.DB.QueryContext(ctx, query, userID)
	if err != nil {
		log.Printf("Failed to query pull requests for user %s: %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pullRequests []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		var statusStr string
		if err := rows.Scan(&pr.Pull_request_id, &pr.Pull_request_name, &pr.Author_id, &statusStr); err != nil {
			log.Printf("Failed to scan pull request for user %s: %v", userID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		pr.Status = models.PullRequestStatus(statusStr)
		pullRequests = append(pullRequests, pr)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows for pull requests: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := models.UserGetReviewResponse{UserID: userID, PullRequests: pullRequests}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PRCreateHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.PRCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var existingPRID string
	err := h.DB.QueryRowContext(ctx, "SELECT pull_request_id FROM pull_requests WHERE pull_request_id = $1", req.Pull_request_id).Scan(&existingPRID)
	if err == nil {
		sendErrorResponse(w, http.StatusConflict, models.ErrorCodePRExists, "PR with this ID already exists")
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("Failed to check existing PR: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var authorTeamName string
	var authorIsActive bool
	err = h.DB.QueryRowContext(ctx, "SELECT team_name, is_active FROM users WHERE user_id = $1", req.Author_id).Scan(&authorTeamName, &authorIsActive)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Author not found")
		return
	}
	if err != nil {
		log.Printf("Failed to get author details: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	reviewerCandidatesQuery := `
		SELECT user_id
		FROM users
		WHERE team_name = $1 AND is_active = TRUE AND user_id <> $2;
	`
	rows, err := h.DB.QueryContext(ctx, reviewerCandidatesQuery, authorTeamName, req.Author_id)
	if err != nil {
		log.Printf("Failed to get reviewer candidates: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var candidateIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("Failed to scan reviewer candidate: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		candidateIDs = append(candidateIDs, id)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows for reviewer candidates: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	assignedReviewers := pickNRandomStrings(candidateIDs, 2)
	var reviewer1ID, reviewer2ID sql.NullString
	if len(assignedReviewers) > 0 {
		reviewer1ID = sql.NullString{String: assignedReviewers[0], Valid: true}
	}
	if len(assignedReviewers) > 1 {
		reviewer2ID = sql.NullString{String: assignedReviewers[1], Valid: true}
	}

	insertPRQuery := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, reviewer1_id, reviewer2_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		RETURNING pull_request_id, pull_request_name, author_id, status, reviewer1_id, reviewer2_id, created_at;
	`
	var pr models.PullRequest
	var statusStr string
	var dbReviewer1ID, dbReviewer2ID sql.NullString
	var createdAt time.Time

	err = h.DB.QueryRowContext(ctx, insertPRQuery,
		req.Pull_request_id, req.Pull_request_name, req.Author_id, models.PullRequestStatusOpen,
		reviewer1ID, reviewer2ID).Scan(
		&pr.Pull_request_id, &pr.Pull_request_name, &pr.Author_id, &statusStr,
		&dbReviewer1ID, &dbReviewer2ID, &createdAt,
	)
	if err != nil {
		log.Printf("Failed to insert new PR: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pr.Status = models.PullRequestStatus(statusStr)
	if dbReviewer1ID.Valid {
		pr.Assigned_reviewers = append(pr.Assigned_reviewers, dbReviewer1ID.String)
	}
	if dbReviewer2ID.Valid {
		pr.Assigned_reviewers = append(pr.Assigned_reviewers, dbReviewer2ID.String)
	}
	pr.CreatedAt = &createdAt

	resp := models.PRResponse{PullRequest: pr}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PRMergeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.PRMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE pull_requests
		SET status = $1, merged_at = CURRENT_TIMESTAMP
		WHERE pull_request_id = $2
		RETURNING pull_request_id, pull_request_name, author_id, status, reviewer1_id, reviewer2_id, created_at, merged_at;
	`
	var pr models.PullRequest
	var statusStr string
	var dbReviewer1ID, dbReviewer2ID sql.NullString
	var createdAt, mergedAt sql.NullTime

	err := h.DB.QueryRowContext(ctx, query, models.PullRequestStatusMerged, req.Pull_request_id).Scan(
		&pr.Pull_request_id, &pr.Pull_request_name, &pr.Author_id, &statusStr,
		&dbReviewer1ID, &dbReviewer2ID, &createdAt, &mergedAt,
	)

	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Pull Request not found")
		return
	}
	if err != nil {
		log.Printf("Failed to merge PR %s: %v", req.Pull_request_id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pr.Status = models.PullRequestStatus(statusStr)
	if dbReviewer1ID.Valid {
		pr.Assigned_reviewers = append(pr.Assigned_reviewers, dbReviewer1ID.String)
	}
	if dbReviewer2ID.Valid {
		pr.Assigned_reviewers = append(pr.Assigned_reviewers, dbReviewer2ID.String)
	}
	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	resp := models.PRResponse{PullRequest: pr}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PRReassignHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req models.PRReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var currentPR models.PullRequest
	var statusStr string
	var authorID string
	var reviewer1ID, reviewer2ID sql.NullString
	var createdAt, mergedAt sql.NullTime

	queryPR := `SELECT pull_request_id, pull_request_name, author_id, status, reviewer1_id, reviewer2_id, created_at, merged_at FROM pull_requests WHERE pull_request_id = $1;`
	err := h.DB.QueryRowContext(ctx, queryPR, req.Pull_request_id).Scan(
		&currentPR.Pull_request_id, &currentPR.Pull_request_name, &authorID, &statusStr,
		&reviewer1ID, &reviewer2ID, &createdAt, &mergedAt,
	)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Pull Request not found")
		return
	}
	if err != nil {
		log.Printf("Failed to get PR details for reassign: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	currentPR.Status = models.PullRequestStatus(statusStr)
	currentPR.Author_id = authorID
	if reviewer1ID.Valid {
		currentPR.Assigned_reviewers = append(currentPR.Assigned_reviewers, reviewer1ID.String)
	}
	if reviewer2ID.Valid {
		currentPR.Assigned_reviewers = append(currentPR.Assigned_reviewers, reviewer2ID.String)
	}
	if createdAt.Valid {
		currentPR.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		currentPR.MergedAt = &mergedAt.Time
	}

	if currentPR.Status == models.PullRequestStatusMerged {
		sendErrorResponse(w, http.StatusConflict, models.ErrorCodePRMerged, "Cannot reassign on merged PR")
		return
	}

	isReviewer1 := reviewer1ID.Valid && reviewer1ID.String == req.Old_reviewer_id
	isReviewer2 := reviewer2ID.Valid && reviewer2ID.String == req.Old_reviewer_id

	if !isReviewer1 && !isReviewer2 {
		sendErrorResponse(w, http.StatusConflict, models.ErrorCodeNotAssigned, "Reviewer is not assigned to this PR")
		return
	}

	var oldReviewerTeamName string
	err = h.DB.QueryRowContext(ctx, "SELECT team_name FROM users WHERE user_id = $1", req.Old_reviewer_id).Scan(&oldReviewerTeamName)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Old reviewer not found")
		return
	}
	if err != nil {
		log.Printf("Failed to get old reviewer's team: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	excludeIDs := map[string]struct{}{
		authorID:            {},
		req.Old_reviewer_id: {},
	}
	if isReviewer1 && reviewer2ID.Valid {
		excludeIDs[reviewer2ID.String] = struct{}{}
	}
	if isReviewer2 && reviewer1ID.Valid {
		excludeIDs[reviewer1ID.String] = struct{}{}
	}

	candidatesQuery := `SELECT user_id FROM users WHERE team_name = $1 AND is_active = TRUE;`
	rows, err := h.DB.QueryContext(ctx, candidatesQuery, oldReviewerTeamName)
	if err != nil {
		log.Printf("Failed to get candidates for reassign: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var allTeamCandidates []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("Failed to scan candidate for reassign: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		allTeamCandidates = append(allTeamCandidates, id)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows for reassign candidates: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var eligibleCandidates []string
	for _, id := range allTeamCandidates {
		if _, excluded := excludeIDs[id]; !excluded {
			eligibleCandidates = append(eligibleCandidates, id)
		}
	}

	if len(eligibleCandidates) == 0 {
		sendErrorResponse(w, http.StatusConflict, models.ErrorCodeNoCandidate, "No active replacement candidate in team")
		return
	}

	newReviewerID := pickNRandomStrings(eligibleCandidates, 1)[0]
	replacedBy := newReviewerID

	var updateQuery string
	if isReviewer1 {
		updateQuery = `UPDATE pull_requests SET reviewer1_id = $1 WHERE pull_request_id = $2;`
	} else {
		updateQuery = `UPDATE pull_requests SET reviewer2_id = $1 WHERE pull_request_id = $2;`
	}

	_, err = h.DB.ExecContext(ctx, updateQuery, newReviewerID, req.Pull_request_id)
	if err != nil {
		log.Printf("Failed to update reviewer ID for PR %s: %v", req.Pull_request_id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = h.DB.QueryRowContext(ctx, queryPR, req.Pull_request_id).Scan(
		&currentPR.Pull_request_id, &currentPR.Pull_request_name, &authorID, &statusStr,
		&reviewer1ID, &reviewer2ID, &createdAt, &mergedAt,
	)
	if err != nil {
		log.Printf("Failed to re-query PR after reassign: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	currentPR.Status = models.PullRequestStatus(statusStr)
	currentPR.Author_id = authorID
	currentPR.Assigned_reviewers = []string{}
	if reviewer1ID.Valid {
		currentPR.Assigned_reviewers = append(currentPR.Assigned_reviewers, reviewer1ID.String)
	}
	if reviewer2ID.Valid {
		currentPR.Assigned_reviewers = append(currentPR.Assigned_reviewers, reviewer2ID.String)
	}
	if createdAt.Valid {
		currentPR.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		currentPR.MergedAt = &mergedAt.Time
	}

	resp := models.PRReassignResponse{
		PullRequest: currentPR,
		Replaced_By: replacedBy,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
