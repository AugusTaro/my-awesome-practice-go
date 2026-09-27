package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/AugusTaro/my-awesome-practice-go/internal/adapter/web"
	"github.com/AugusTaro/my-awesome-practice-go/internal/todo"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "todo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS todos(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)
`)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	todostore := todo.NewStore()
	todoHandler := web.NewHandler(todostore)
	mux.HandleFunc("POST /todos", todoHandler.CreateTodo)
	mux.HandleFunc("GET /todos", todoHandler.GetTodos)
	http.ListenAndServe(":8080", mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK\n")
}
