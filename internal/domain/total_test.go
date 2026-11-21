package domain_test

import (
	"math"
	"testing"

	"pos/internal/domain"
)

func TestWeightedTotalUsesCurrentRubricAndEvaluations(t *testing.T) {
	var book domain.Gradebook
	must(t, book.AddStudent("s", "Анна"))
	zero := domain.NewCalculator(domain.MissingAsZero{})
	ignore := domain.NewCalculator(domain.IgnoreMissing{})
	assertTotal(t, zero, &book, 0, 0, 0, false)

	must(t, book.AddCriterion("exam", "Экзамен", 20, 3))
	must(t, book.AddCriterion("project", "Проект", 10, 1))
	assertTotal(t, zero, &book, 0, 0, 2, true)
	assertTotal(t, ignore, &book, 0, 0, 2, false)

	must(t, book.Assign("s", "exam", 10))
	assertTotal(t, zero, &book, 37.5, 1, 2, true)
	assertTotal(t, ignore, &book, 50, 1, 2, true)

	must(t, book.Assign("s", "project", 0))
	assertTotal(t, ignore, &book, 37.5, 2, 2, true)
	must(t, book.Correct("s", "project", 10))
	assertTotal(t, zero, &book, 62.5, 2, 2, true)
	must(t, book.Cancel("s", "exam"))
	assertTotal(t, zero, &book, 25, 1, 2, true)
	assertTotal(t, ignore, &book, 100, 1, 2, true)

	must(t, book.AddCriterion("report", "Отчёт", 5, 1))
	assertTotal(t, zero, &book, 20, 1, 3, true)
	if _, err := zero.Totals(&book, "unknown"); err == nil {
		t.Fatal("итог рассчитан для неизвестного студента")
	}
}

func TestWeightedTotalWithExtremeWeights(t *testing.T) {
	var book domain.Gradebook
	must(t, book.AddStudent("s", "Анна"))
	must(t, book.AddCriterion("a", "Первый", math.MaxFloat64, math.MaxFloat64))
	must(t, book.AddCriterion("b", "Второй", 1, math.MaxFloat64))
	must(t, book.AddCriterion("c", "Третий", 1, math.SmallestNonzeroFloat64))
	must(t, book.Assign("s", "a", math.MaxFloat64))
	assertTotal(t, domain.Calculator{}, &book, 50, 1, 3, true)
	must(t, book.Cancel("s", "a"))
	must(t, book.Assign("s", "c", 1))
	assertTotal(t, domain.NewCalculator(domain.IgnoreMissing{}), &book, 100, 1, 3, true)
}

func assertTotal(t *testing.T, calculator domain.Calculator, book *domain.Gradebook, percent float64, graded, criteria int, available bool) {
	t.Helper()
	rows, err := calculator.Totals(book, "s")
	must(t, err)
	if len(rows) != 1 {
		t.Fatalf("ожидался один итог, получено %d", len(rows))
	}
	row := rows[0]
	if row.Student.ID() != "s" || row.Available != available || row.Graded != graded ||
		row.Criteria != criteria || math.IsNaN(row.Percent) || math.Abs(row.Percent-percent) > 1e-9 {
		t.Fatalf("итог %+v, ожидались %g%%, %d/%d, доступен=%t", row, percent, graded, criteria, available)
	}
}
