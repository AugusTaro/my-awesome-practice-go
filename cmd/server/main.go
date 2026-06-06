package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("POST /todos", CreateTodoHandler)
	mux.HandleFunc("GET /todos", GetTodosHandler)
	http.ListenAndServe(":8080", mux)
}

type Todo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type createTodoRequest struct {
	Name string `json:"name"`
}

var todos []Todo
var nextID = 1

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK\n")
}
func CreateTodoHandler(w http.ResponseWriter, r *http.Request) {
	var req createTodoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	todo := Todo{
		ID:   nextID,
		Name: req.Name,
	}
	todos = append(todos, todo)
	nextID++
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}
func GetTodosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}
