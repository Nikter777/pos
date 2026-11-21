package domain_test

import (
	"math"
	"slices"
	"testing"

	"pos/internal/domain"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationInvariants(t *testing.T) {
	var book domain.Gradebook
	for _, input := range [][2]string{{"", "Имя"}, {"   ", "Имя"}, {"id with spaces", "Имя"}, {"id\u00a0spaces", "Имя"}, {"id", " \t "}} {
		if err := book.AddStudent(input[0], input[1]); err == nil {
			t.Errorf("принят студент с полями %q", input)
		}
		if err := book.AddCriterion(input[0], input[1], 10, 1); err == nil {
			t.Errorf("принят критерий с полями %q", input)
		}
	}
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := book.AddCriterion("id", "Критерий", value, 1); err == nil {
			t.Errorf("принят максимум %g", value)
		}
		if err := book.AddCriterion("id", "Критерий", 10, value); err == nil {
			t.Errorf("принят вес %g", value)
		}
	}
	if len(book.Students()) != 0 || len(book.Criteria()) != 0 {
		t.Fatal("отклонённые записи попали в ведомость")
	}

	must(t, book.AddStudent(" s2 ", " Борис "))
	must(t, book.AddStudent("s1", "Анна"))
	must(t, book.AddCriterion(" c2 ", " Проект ", 20, 3))
	must(t, book.AddCriterion("c1", "Тест", 10, 1))
	if err := book.AddStudent("s2", "Другое имя"); err == nil {
		t.Fatal("принят дубликат студента")
	}
	if err := book.AddCriterion("c2", "Другое название", 5, 1); err == nil {
		t.Fatal("принят дубликат критерия")
	}
	students, criteria := book.Students(), book.Criteria()
	if len(students) != 2 || students[0].ID() != "s1" || students[1].Name() != "Борис" {
		t.Fatalf("студенты: %+v", students)
	}
	if len(criteria) != 2 || criteria[0].ID() != "c1" || criteria[1].Name() != "Проект" ||
		criteria[1].MaxScore() != 20 || criteria[1].Weight() != 3 {
		t.Fatalf("критерии: %+v", criteria)
	}
}

func TestEvaluationLifecycle(t *testing.T) {
	var book domain.Gradebook
	must(t, book.AddStudent("s", "Анна"))
	must(t, book.AddStudent("other", "Борис"))
	must(t, book.AddCriterion("b", "Проект", 10, 1))
	must(t, book.AddCriterion("a", "Тест", 10, 1))

	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{"неизвестный студент", func() error { return book.Assign("unknown", "a", 1) }},
		{"неизвестный критерий", func() error { return book.Assign("s", "unknown", 1) }},
		{"исправление отсутствующей", func() error { return book.Correct("s", "a", 1) }},
		{"отмена по неизвестному критерию", func() error { return book.Cancel("s", "unknown") }},
		{"отмена отсутствующей", func() error { return book.Cancel("s", "a") }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); err == nil {
				t.Fatal("операция должна быть отклонена")
			}
		})
	}
	if _, err := book.Evaluations("unknown"); err == nil {
		t.Fatal("возвращены оценки неизвестного студента")
	}
	for _, points := range []float64{-1, 10.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := book.Assign("s", "a", points); err == nil {
			t.Errorf("принят балл %g", points)
		}
	}

	must(t, book.Assign("s", "b", 10))
	must(t, book.Assign(" s ", " a ", 0))
	if err := book.Assign("s", "a", 5); err == nil {
		t.Fatal("повторное выставление изменило существующую оценку")
	}
	if err := book.Correct("s", "a", 11); err == nil {
		t.Fatal("исправление в обход максимума")
	}
	assertEvaluations(t, &book, "s", []string{"a", "b"}, []float64{0, 10})
	assertEvaluations(t, &book, "other", nil, nil)

	must(t, book.Correct("s", "a", 7.5))
	must(t, book.Cancel("s", "b"))
	assertEvaluations(t, &book, "s", []string{"a"}, []float64{7.5})
	must(t, book.Assign("s", "b", 9))
	assertEvaluations(t, &book, "s", []string{"a", "b"}, []float64{7.5, 9})
}

func assertEvaluations(t *testing.T, book *domain.Gradebook, student string, wantIDs []string, wantScores []float64) {
	t.Helper()
	rows, err := book.Evaluations(student)
	must(t, err)
	var ids []string
	var scores []float64
	for _, row := range rows {
		ids = append(ids, row.CriterionID())
		scores = append(scores, row.Score().Points())
	}
	if !slices.Equal(ids, wantIDs) || !slices.Equal(scores, wantScores) {
		t.Fatalf("оценки %v %v, ожидались %v %v", ids, scores, wantIDs, wantScores)
	}
}
