package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"pr_reviewer_assignment_service/config"
	"pr_reviewer_assignment_service/handler"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

//go:embed database/databaseInit.sql
var initialSchemaSQL []byte

var db *sql.DB

func main() {
	conf := config.LoadConfig()

	var err error
	db, err = sql.Open("postgres", conf.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверка соединения с бд
	if err := db.PingContext(initCtx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Println("Connected to PostgreSQL")
	// Окончание проверки

	// Применение начальной схемы БД
	log.Println("Applying initial database schema...")
	err = applyInitialSchema(initCtx, db, string(initialSchemaSQL))
	if err != nil {
		log.Fatalf("Failed to apply initial database schema: %v", err)
	}
	log.Println("Initial database schema applied successfully or already exists.")
	// Окончание применения схемы

	Handler := handler.NewHandler(conf, db)
	router := mux.NewRouter()

	router.HandleFunc("/team/add", Handler.TeamAddHandler).Methods("POST")
	router.HandleFunc("/team/get", Handler.TeamGetHandler).Methods("GET")
	router.HandleFunc("/users/setIsActive", Handler.UserSetIsActiveHandler).Methods("POST")
	router.HandleFunc("/users/getReview", Handler.UserGetReviewHandler).Methods("GET")
	router.HandleFunc("/pullRequest/create", Handler.PRCreateHandler).Methods("POST")
	router.HandleFunc("/pullRequest/merge", Handler.PRMergeHandler).Methods("POST")
	router.HandleFunc("/pullRequest/reassign", Handler.PRReassignHandler).Methods("POST")
	router.HandleFunc("/stats/overall", Handler.GetOverallStatsHandler).Methods("GET") //эндпоинт для ствтистики

	serverAddress := ":" + conf.Port
	fmt.Printf("Service listening on %s\n", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, router))
}

func applyInitialSchema(ctx context.Context, db *sql.DB, schema string) error {
	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute initial schema SQL: %w", err)
	}
	return nil
}
