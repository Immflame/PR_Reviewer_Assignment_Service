package models

import "time"

type ErrorCode string

const (
	ErrorCodeTeamExists  ErrorCode = "TEAM_EXISTS"
	ErrorCodePRExists    ErrorCode = "PR_EXISTS"
	ErrorCodePRMerged    ErrorCode = "PR_MERGED"
	ErrorCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	ErrorCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	ErrorCodeNotFound    ErrorCode = "NOT_FOUND"
)

type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
)

type ErrorDetails struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type UserSetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type UserSetIsActiveResponse struct {
	User User `json:"user"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type TeamAddResponse struct {
	Team Team `json:"team"`
}

type PullRequest struct {
	Pull_request_id    string            `json:"pull_request_id"`
	Pull_request_name  string            `json:"pull_request_name"`
	Author_id          string            `json:"author_id"`
	Status             PullRequestStatus `json:"status"`
	Assigned_reviewers []string          `json:"assigned_reviewers"`
	CreatedAt          *time.Time        `json:"createdAt,omitempty"`
	MergedAt           *time.Time        `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	Pull_request_id   string            `json:"pull_request_id"`
	Pull_request_name string            `json:"pull_request_name"`
	Author_id         string            `json:"author_id"`
	Status            PullRequestStatus `json:"status"`
}

type PRCreateRequest struct {
	Pull_request_id   string `json:"pull_request_id"`
	Pull_request_name string `json:"pull_request_name"`
	Author_id         string `json:"author_id"`
}

type PRResponse struct {
	PullRequest PullRequest `json:"pr"`
}

type PRMergeRequest struct {
	Pull_request_id string `json:"pull_request_id"`
}

type PRReassignRequest struct {
	Pull_request_id string `json:"pull_request_id"`
	Old_reviewer_id string `json:"old_reviewer_id"`
}

type PRReassignResponse struct {
	PullRequest PullRequest `json:"pr"`
	Replaced_By string      `json:"replaced_by"`
}

type UserGetReviewResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type UserReviewStats struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	ReviewsAssigned  int    `json:"reviews_assigned"`
	ReviewsCompleted int    `json:"reviews_completed"`
}

type PRStats struct {
	TotalPRs          int     `json:"total_prs"`
	OpenPRs           int     `json:"open_prs"`
	MergedPRs         int     `json:"merged_prs"`
	AvgReviewersPerPR float64 `json:"avg_reviewers_per_pr"`
}

type OverallStatsResponse struct {
	UserStats []UserReviewStats `json:"user_stats"`
	PRStats   PRStats           `json:"pr_stats"`
}
