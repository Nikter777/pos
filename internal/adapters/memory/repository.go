package memory

import (
	"sync"

	"pos/internal/domain"
)

type Repository struct {
	mu   sync.RWMutex
	book *domain.Gradebook
}

func NewRepository() *Repository {
	return &Repository{book: domain.NewGradebook()}
}

func (r *Repository) Snapshot() *domain.Gradebook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.book.Clone()
}

func (r *Repository) Update(change func(*domain.Gradebook) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	next := r.book.Clone()
	if err := change(next); err != nil {
		return err
	}
	r.book = next.Clone()
	return nil
}
