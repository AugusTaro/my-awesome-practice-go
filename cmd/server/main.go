package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

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

	var store = TodoStore{
		nextID: 1,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		CreateTodoHandler(w, r, &store)
	})
	mux.HandleFunc("GET /todos", func(w http.ResponseWriter, r *http.Request) {
		GetTodosHandler(w, r, &store)
	})
	http.ListenAndServe(":8080", mux)
}

type Todo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type createTodoRequest struct {
	Name string `json:"name"`
}
type TodoStore struct {
	todos  []Todo
	nextID int
}

func (s *TodoStore) Add(name string) Todo {
	todo := Todo{
		ID:   s.nextID,
		Name: name,
	}
	s.todos = append(s.todos, todo)
	s.nextID++
	return todo
}

func (s *TodoStore) List() []Todo {
	return s.todos
}
func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK\n")
}
func CreateTodoHandler(w http.ResponseWriter, r *http.Request, s *TodoStore) {
	var req createTodoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	todo := s.Add(req.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}
func GetTodosHandler(w http.ResponseWriter, r *http.Request, s *TodoStore) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.List())
}
