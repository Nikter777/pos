package application

import "pos/internal/domain"

type GradebookRepository interface {
	Snapshot() *domain.Gradebook
	Update(func(*domain.Gradebook) error) error
}

type TotalExporter interface {
	Export(path string, rows []domain.Total) error
}

type Gradebook struct {
	repo       GradebookRepository
	calculator domain.Calculator
	exporter   TotalExporter
}

func NewGradebook(repo GradebookRepository, calculator domain.Calculator, exporter TotalExporter) *Gradebook {
	return &Gradebook{repo: repo, calculator: calculator, exporter: exporter}
}

func (a *Gradebook) AddStudent(id, name string) error {
	return a.repo.Update(func(book *domain.Gradebook) error {
		return book.AddStudent(id, name)
	})
}

func (a *Gradebook) AddCriterion(id, name string, maxScore, weight float64) error {
	return a.repo.Update(func(book *domain.Gradebook) error {
		return book.AddCriterion(id, name, maxScore, weight)
	})
}

func (a *Gradebook) Students() []domain.Student {
	return a.repo.Snapshot().Students()
}

func (a *Gradebook) Criteria() []domain.Criterion {
	return a.repo.Snapshot().Criteria()
}

func (a *Gradebook) Evaluations(studentID string) ([]domain.Evaluation, error) {
	return a.repo.Snapshot().Evaluations(studentID)
}

func (a *Gradebook) Assign(studentID, criterionID string, points float64) error {
	return a.repo.Update(func(book *domain.Gradebook) error {
		return book.Assign(studentID, criterionID, points)
	})
}

func (a *Gradebook) Correct(studentID, criterionID string, points float64) error {
	return a.repo.Update(func(book *domain.Gradebook) error {
		return book.Correct(studentID, criterionID, points)
	})
}

func (a *Gradebook) Cancel(studentID, criterionID string) error {
	return a.repo.Update(func(book *domain.Gradebook) error {
		return book.Cancel(studentID, criterionID)
	})
}

func (a *Gradebook) Totals(studentID string) ([]domain.Total, error) {
	return a.calculator.Totals(a.repo.Snapshot(), studentID)
}

func (a *Gradebook) ExportTotals(path string) error {
	rows, err := a.Totals("")
	if err != nil {
		return err
	}
	return a.exporter.Export(path, rows)
}
