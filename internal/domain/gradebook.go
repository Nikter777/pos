package domain

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"unicode"
)

type registration struct {
	id   string
	name string
}

func (r registration) ID() string   { return r.id }
func (r registration) Name() string { return r.name }

func newRegistration(id, name string) (registration, error) {
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if id == "" || strings.IndexFunc(id, unicode.IsSpace) >= 0 {
		return registration{}, fmt.Errorf("идентификатор должен быть непустым и без пробелов")
	}
	if name == "" {
		return registration{}, fmt.Errorf("название или имя обязательно")
	}
	return registration{id: id, name: name}, nil
}

type Student struct{ registration }

type Criterion struct {
	registration
	maxScore float64
	weight   float64
}

func (c Criterion) MaxScore() float64 { return c.maxScore }
func (c Criterion) Weight() float64   { return c.weight }

type Score struct{ points float64 }

func (s Score) Points() float64 { return s.points }

type evaluationKey struct {
	studentID   string
	criterionID string
}

type Evaluation struct {
	key   evaluationKey
	score Score
}

func (e Evaluation) CriterionID() string { return e.key.criterionID }
func (e Evaluation) Score() Score        { return e.score }

type Gradebook struct {
	students map[string]Student
	criteria map[string]Criterion
	scores   map[evaluationKey]Score
}

func (g *Gradebook) Clone() *Gradebook {
	return &Gradebook{
		students: maps.Clone(g.students),
		criteria: maps.Clone(g.criteria),
		scores:   maps.Clone(g.scores),
	}
}

func (g *Gradebook) AddStudent(id, name string) error {
	r, err := newRegistration(id, name)
	if err != nil {
		return err
	}
	if _, exists := g.students[r.id]; exists {
		return fmt.Errorf("студент %q уже существует", r.id)
	}
	if g.students == nil {
		g.students = make(map[string]Student)
	}
	g.students[r.id] = Student{registration: r}
	return nil
}

func (g *Gradebook) AddCriterion(id, name string, maxScore, weight float64) error {
	r, err := newRegistration(id, name)
	if err != nil {
		return err
	}
	if _, exists := g.criteria[r.id]; exists {
		return fmt.Errorf("критерий %q уже существует", r.id)
	}
	if !finite(maxScore) || maxScore <= 0 {
		return fmt.Errorf("максимальный балл должен быть конечным и больше нуля")
	}
	if !finite(weight) || weight <= 0 {
		return fmt.Errorf("вес должен быть конечным и больше нуля")
	}
	if g.criteria == nil {
		g.criteria = make(map[string]Criterion)
	}
	g.criteria[r.id] = Criterion{registration: r, maxScore: maxScore, weight: weight}
	return nil
}

func (g *Gradebook) Student(id string) (Student, error) {
	student, exists := g.students[strings.TrimSpace(id)]
	if !exists {
		return Student{}, fmt.Errorf("студент %q не найден", id)
	}
	return student, nil
}

func (g *Gradebook) Students() []Student {
	list := make([]Student, 0, len(g.students))
	for _, student := range g.students {
		list = append(list, student)
	}
	slices.SortFunc(list, func(a, b Student) int { return strings.Compare(a.ID(), b.ID()) })
	return list
}

func (g *Gradebook) Criteria() []Criterion {
	list := make([]Criterion, 0, len(g.criteria))
	for _, criterion := range g.criteria {
		list = append(list, criterion)
	}
	slices.SortFunc(list, func(a, b Criterion) int { return strings.Compare(a.ID(), b.ID()) })
	return list
}

func (g *Gradebook) Evaluations(studentID string) ([]Evaluation, error) {
	student, err := g.Student(studentID)
	if err != nil {
		return nil, err
	}
	var list []Evaluation
	for key, score := range g.scores {
		if key.studentID == student.ID() {
			list = append(list, Evaluation{key: key, score: score})
		}
	}
	slices.SortFunc(list, func(a, b Evaluation) int {
		return strings.Compare(a.CriterionID(), b.CriterionID())
	})
	return list, nil
}

func (g *Gradebook) Assign(studentID, criterionID string, points float64) error {
	evaluation, err := g.newEvaluation(studentID, criterionID, points)
	if err != nil {
		return err
	}
	if _, exists := g.scores[evaluation.key]; exists {
		return fmt.Errorf("оценка уже выставлена, используйте исправление")
	}
	if g.scores == nil {
		g.scores = make(map[evaluationKey]Score)
	}
	g.scores[evaluation.key] = evaluation.score
	return nil
}

func (g *Gradebook) Correct(studentID, criterionID string, points float64) error {
	evaluation, err := g.newEvaluation(studentID, criterionID, points)
	if err != nil {
		return err
	}
	if _, exists := g.scores[evaluation.key]; !exists {
		return fmt.Errorf("оценка ещё не выставлена")
	}
	g.scores[evaluation.key] = evaluation.score
	return nil
}

func (g *Gradebook) Cancel(studentID, criterionID string) error {
	key, _, err := g.resolveEvaluation(studentID, criterionID)
	if err != nil {
		return err
	}
	if _, exists := g.scores[key]; !exists {
		return fmt.Errorf("оценка ещё не выставлена")
	}
	delete(g.scores, key)
	return nil
}

func (g *Gradebook) newEvaluation(studentID, criterionID string, points float64) (Evaluation, error) {
	key, criterion, err := g.resolveEvaluation(studentID, criterionID)
	if err != nil {
		return Evaluation{}, err
	}
	if !finite(points) || points < 0 || points > criterion.MaxScore() {
		return Evaluation{}, fmt.Errorf("балл должен быть конечным и лежать в диапазоне [0, %g]", criterion.MaxScore())
	}
	return Evaluation{key: key, score: Score{points: points}}, nil
}

func (g *Gradebook) resolveEvaluation(studentID, criterionID string) (evaluationKey, Criterion, error) {
	student, err := g.Student(studentID)
	if err != nil {
		return evaluationKey{}, Criterion{}, err
	}
	criterion, exists := g.criteria[strings.TrimSpace(criterionID)]
	if !exists {
		return evaluationKey{}, Criterion{}, fmt.Errorf("критерий %q отсутствует в рубрикаторе", criterionID)
	}
	return evaluationKey{studentID: student.ID(), criterionID: criterion.ID()}, criterion, nil
}

func finite(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, 0)
}
