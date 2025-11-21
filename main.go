package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	config := LoadConfig()

	var err error
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	//проверка соединения с бд
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Println("Connected to PostgreSQL")
	//окончание проверки

	Handler := NewHandler(config, db)
	router := mux.NewRouter()

	router.HandleFunc("/team/add", Handler.TeamAddHandler).Methods("POST")
	router.HandleFunc("/team/get", Handler.TeamGetHandler).Methods("GET")
	router.HandleFunc("/users/setIsActive", Handler.UserSetIsActiveHandler).Methods("POST")
	router.HandleFunc("/users/getReview", Handler.UsersGetReviewHandler).Methods("GET")
	router.HandleFunc("/pullRequest/create", Handler.PRCreateHandler).Methods("POST")
	router.HandleFunc("/pullRequest/merge", Handler.PRMergeHandler).Methods("POST")
	router.HandleFunc("/pullRequest/reassign", Handler.PRReassignHandler).Methods("POST")

	serverAddress := ":" + config.Port
	fmt.Printf("Service listening on %s\n", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, router))
}
