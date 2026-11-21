package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
)

func TestConsoleGradebookAndExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "итоги курса.csv")
	input := strings.Join([]string{
		"", "help", "students", "criteria", "total",
		"student-add s2 Борис Петров", "student-add\ts1\tАнна, Иванова",
		"students", "total s2",
		"criterion-add exam 20 3 Экзамен", "criterion-add project 10 1 Групповой проект",
		"criteria", "grades s1",
		"grade-set s1 exam 10", "grade-set s1 project 10", "grades s1", "total s1",
		"grade-edit s1 exam 20", "total s1", "grade-cancel s1 project", "grades s1", "total",
		"export " + path, "export " + path, "exit", "unreachable",
	}, "\n")
	output, diagnostic, status := runSession(nil, input)
	if status != 0 || diagnostic != "" {
		t.Fatalf("код %d, stderr %q", status, diagnostic)
	}
	for _, want := range []string{
		"grade-edit СТУДЕНТ КРИТЕРИЙ БАЛЛ", "Студентов пока нет.", "Рубрикатор пуст.",
		"s1\tАнна, Иванова\ns2\tБорис Петров", "итог не определён\tоценено 0/0",
		"project\tГрупповой проект\tмаксимум 10\tвес 1", "Оценок пока нет.",
		"exam\t10\nproject\t10", "62.50%\tоценено 2/2", "100.00%\tоценено 2/2",
		"s1\tАнна, Иванова\t75.00%\tоценено 1/2\ns2\tБорис Петров\t0.00%\tоценено 0/2",
		"Итоги экспортированы:", "Ошибка: создание CSV",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("нет %q в выводе:\n%s", want, output)
		}
	}
	if strings.Contains(output, "unreachable") {
		t.Fatal("команда выполнена после exit")
	}
	assertCSV(t, path, [][]string{
		{"student_id", "student_name", "total_percent", "graded", "criteria"},
		{"s1", "Анна, Иванова", "75.00", "1", "2"},
		{"s2", "Борис Петров", "0.00", "0", "2"},
	})
}

func TestConsoleRejectsInvalidCommandsAndContinues(t *testing.T) {
	tests := []struct{ command, message string }{
		{"student-add s", "формат: student-add"},
		{"student-add s Дубликат", "уже существует"},
		{"criterion-add c 10", "формат: criterion-add"},
		{"criterion-add c2 нет 1 Тест", "максимальный балл:"},
		{"criterion-add c2 10 нет Тест", "вес:"},
		{"criterion-add c2 10 Inf Тест", "вес должен быть конечным"},
		{"grade-set s c", "формат: grade-set"},
		{"grade-set s c нет", "балл: не удалось прочитать число"},
		{"grade-set s c NaN", "балл должен быть конечным"},
		{"grade-set s other 1", "отсутствует в рубрикаторе"},
		{"grade-set other c 1", "студент \"other\" не найден"},
		{"grade-set s c 11", "диапазоне [0, 10]"},
		{"grade-set s c 1", "оценка уже выставлена"},
		{"grade-edit s c -1", "диапазоне [0, 10]"},
		{"grade-cancel s", "формат: grade-cancel"},
		{"grade-cancel s missing", "отсутствует в рубрикаторе"},
		{"grades", "формат: grades"},
		{"grades other", "студент \"other\" не найден"},
		{"total s c", "формат: total"},
		{"total other", "студент \"other\" не найден"},
		{"export", "укажите путь"},
		{"help extra", "не принимает аргументы"},
		{"exit extra", "не принимает аргументы"},
		{"other", "неизвестная команда"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			input := "student-add s Анна\ncriterion-add c 10 1 Тест\ngrade-set s c 5\n" +
				tt.command + "\ntotal s\n"
			output, diagnostic, status := runSession(nil, input)
			_, errorLine, _ := strings.Cut(output, "Ошибка: ")
			message, _, _ := strings.Cut(errorLine, "\n")
			if status != 0 || diagnostic != "" || !strings.Contains(message, tt.message) ||
				!strings.Contains(output, "50.00%\tоценено 1/1") {
				t.Fatalf("код %d, stderr %q, вывод %q", status, diagnostic, output)
			}
		})
	}
}

func TestConsoleMissingStrategyAndUndefinedCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignore.csv")
	input := fmt.Sprintf("student-add s Анна\nstudent-add t Борис\ncriterion-add a 10 1 Тест\n"+
		"criterion-add b 20 3 Проект\ngrade-set s a 5\ntotal\nexport %s\n", path)
	output, diagnostic, status := runSession([]string{"-missing=ignore"}, input)
	if status != 0 || diagnostic != "" ||
		!strings.Contains(output, "50.00%\tоценено 1/2") ||
		!strings.Contains(output, "итог не определён\tоценено 0/2") {
		t.Fatalf("код %d, stderr %q, вывод %q", status, diagnostic, output)
	}
	assertCSV(t, path, [][]string{
		{"student_id", "student_name", "total_percent", "graded", "criteria"},
		{"s", "Анна", "50.00", "1", "2"},
		{"t", "Борис", "", "0", "2"},
	})
}

func TestEntryPointFlagsAndIOErrors(t *testing.T) {
	for _, tt := range []struct {
		arg     string
		status  int
		message string
	}{
		{"-help", 0, "missing"},
		{"-missing=other", 2, "неизвестная политика"},
		{"unexpected", 2, "Неожиданные аргументы"},
		{"-unexpected", 2, "flag provided but not defined"},
	} {
		t.Run(tt.arg, func(t *testing.T) {
			_, diagnostic, status := runSession([]string{tt.arg}, "")
			if status != tt.status || !strings.Contains(diagnostic, tt.message) {
				t.Fatalf("код %d, stderr %q", status, diagnostic)
			}
		})
	}
	var diagnostic bytes.Buffer
	failure := iotest.ErrReader(errors.New("сломанный ввод"))
	if status := run(nil, failure, &bytes.Buffer{}, &diagnostic); status != 1 ||
		!strings.Contains(diagnostic.String(), "сломанный ввод") {
		t.Fatalf("код %d, stderr %q", status, diagnostic.String())
	}
	diagnostic.Reset()
	if status := run(nil, strings.NewReader(""), brokenWriter{}, &diagnostic); status != 1 ||
		!strings.Contains(diagnostic.String(), "сломанный вывод") {
		t.Fatalf("код %d, stderr %q", status, diagnostic.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("сломанный вывод")
}

func runSession(args []string, input string) (string, string, int) {
	var output, diagnostic bytes.Buffer
	status := run(args, strings.NewReader(input), &output, &diagnostic)
	return output.String(), diagnostic.String(), status
}

func assertCSV(t *testing.T, path string, want [][]string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(rows, want, slices.Equal[[]string]) {
		t.Fatalf("CSV %q, ожидалось %q", rows, want)
	}
}
