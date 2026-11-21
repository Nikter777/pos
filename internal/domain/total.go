package domain

import "math"

type MissingPolicy interface {
	includeMissing() bool
}

type MissingAsZero struct{}

func (MissingAsZero) includeMissing() bool { return true }

type IgnoreMissing struct{}

func (IgnoreMissing) includeMissing() bool { return false }

type Calculator struct {
	missing MissingPolicy
}

func NewCalculator(missing MissingPolicy) Calculator {
	return Calculator{missing: missing}
}

type Total struct {
	Student   Student
	Percent   float64
	Available bool
	Graded    int
	Criteria  int
}

func (c Calculator) Totals(book *Gradebook, studentID string) ([]Total, error) {
	var students []Student
	if studentID == "" {
		students = book.Students()
	} else {
		student, err := book.Student(studentID)
		if err != nil {
			return nil, err
		}
		students = []Student{student}
	}

	criteria := book.Criteria()
	includeMissing := c.missing == nil || c.missing.includeMissing()
	results := make([]Total, 0, len(students))
	for _, student := range students {
		results = append(results, calculateTotal(book, student, criteria, includeMissing))
	}
	return results, nil
}

func calculateTotal(book *Gradebook, student Student, criteria []Criterion, includeMissing bool) Total {
	total := Total{Student: student, Criteria: len(criteria)}
	var maxWeight float64
	for _, criterion := range criteria {
		_, graded := book.scores[evaluationKey{student.ID(), criterion.ID()}]
		if graded {
			total.Graded++
		}
		if graded || includeMissing {
			maxWeight = max(maxWeight, criterion.Weight())
		}
	}
	if maxWeight == 0 {
		return total
	}

	var numerator, denominator float64
	for _, criterion := range criteria {
		score, graded := book.scores[evaluationKey{student.ID(), criterion.ID()}]
		if !graded && !includeMissing {
			continue
		}
		weight := criterion.Weight() / maxWeight
		numerator += weight * (score.Points() / criterion.MaxScore())
		denominator += weight
	}
	total.Available = true
	total.Percent = math.Min(100, 100*(numerator/denominator))
	return total
}
