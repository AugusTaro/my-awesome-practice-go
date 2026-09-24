package todo

type Store struct {
	todos  []Todo
	nextID int
}

func NewStore() *Store {
	return &Store{
		todos:  []Todo{},
		nextID: 1,
	}
}
func (s *Store) Add(name string) Todo {
	todo := Todo{
		ID:   s.nextID,
		Name: name,
	}
	s.todos = append(s.todos, todo)
	s.nextID++
	return todo
}

func (s *Store) List() []Todo {
	return s.todos
}
