package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
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
