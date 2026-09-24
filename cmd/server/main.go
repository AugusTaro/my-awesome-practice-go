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
	todoHandler := NewTodoHandler(&store)
	mux.HandleFunc("POST /todos", todoHandler.CreateTodo)
	mux.HandleFunc("GET /todos", todoHandler.GetTodos)
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

type TodoHandler struct {
	store *TodoStore
}

func NewTodoHandler(store *TodoStore) *TodoHandler {
	return &TodoHandler{store: store}
}

func (th *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req createTodoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	todo := th.store.Add(req.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}
func (th *TodoHandler) GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(th.store.List())
}
