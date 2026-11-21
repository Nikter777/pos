package memory_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"pos/internal/adapters/memory"
	"pos/internal/domain"
)

func TestUpdateIsAtomicAndSnapshotsAreIsolated(t *testing.T) {
	repo := memory.NewRepository()
	aborted := errors.New("отмена операции")
	err := repo.Update(func(book *domain.Gradebook) error {
		if err := book.AddStudent("discarded", "Анна"); err != nil {
			return err
		}
		return aborted
	})
	if !errors.Is(err, aborted) || len(repo.Snapshot().Students()) != 0 {
		t.Fatalf("операция не откатилась: %v", err)
	}

	var retained *domain.Gradebook
	err = repo.Update(func(book *domain.Gradebook) error {
		retained = book
		return book.AddStudent("saved", "Борис")
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := repo.Snapshot()
	if err := snapshot.AddStudent("snapshot", "Вера"); err != nil {
		t.Fatal(err)
	}
	if err := retained.AddStudent("retained", "Глеб"); err != nil {
		t.Fatal(err)
	}
	if list := repo.Snapshot().Students(); len(list) != 1 || list[0].ID() != "saved" {
		t.Fatalf("внешняя мутация изменила репозиторий: %v", list)
	}
}

func TestConcurrentUpdatesPreserveStudents(t *testing.T) {
	repo := memory.NewRepository()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.Update(func(book *domain.Gradebook) error {
				return book.AddStudent(fmt.Sprintf("s%d", i), "Студент")
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if count := len(repo.Snapshot().Students()); count != 16 {
		t.Fatalf("после обновлений сохранено %d из 16 записей", count)
	}
}
