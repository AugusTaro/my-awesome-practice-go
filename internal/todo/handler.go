package todo

import (
	"encoding/json"
	"net/http"
)

type createTodoRequest struct {
	Name string `json:"name"`
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
