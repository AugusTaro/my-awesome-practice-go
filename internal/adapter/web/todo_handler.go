package web

import (
	"encoding/json"
	"net/http"

	"github.com/AugusTaro/my-awesome-practice-go/internal/todo"
)

type createTodoRequest struct {
	Name string `json:"name"`
}

type Handler struct {
	store *todo.Store
}

func NewHandler(store *todo.Store) *Handler {
	return &Handler{store: store}
}

func (th *Handler) CreateTodo(w http.ResponseWriter, r *http.Request) {
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
func (th *Handler) GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(th.store.List())
}
