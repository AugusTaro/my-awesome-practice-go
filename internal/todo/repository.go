package todo

type TodoStore struct {
	todos  []Todo
	nextID int
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos:  []Todo{},
		nextID: 1,
	}
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
