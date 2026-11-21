package console

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"pos/internal/application"
)

type Console struct {
	gradebook *application.Gradebook
}

func New(gradebook *application.Gradebook) *Console {
	return &Console{gradebook: gradebook}
}

const help = `Команды
  student-add ID ИМЯ
  students
  criterion-add ID МАКСИМУМ ВЕС НАЗВАНИЕ
  criteria
  grade-set СТУДЕНТ КРИТЕРИЙ БАЛЛ
  grade-edit СТУДЕНТ КРИТЕРИЙ БАЛЛ
  grade-cancel СТУДЕНТ КРИТЕРИЙ
  grades СТУДЕНТ
  total [СТУДЕНТ]
  export ПУТЬ
  help
  exit`

func (c *Console) Run(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	out := bufio.NewWriter(output)
	fmt.Fprintln(out, "ПОС. Введите help для списка команд.")
	for {
		fmt.Fprint(out, "> ")
		if err := out.Flush(); err != nil {
			return err
		}
		if !scanner.Scan() {
			return scanner.Err()
		}
		parts := splitLine(scanner.Text(), 2)
		if len(parts) == 0 {
			continue
		}
		command := parts[0]
		var args string
		if len(parts) == 2 {
			args = parts[1]
		}
		if command == "exit" && args == "" {
			return nil
		}
		if err := c.execute(out, command, args); err != nil {
			fmt.Fprintln(out, "Ошибка:", err)
		}
	}
}

func (c *Console) execute(out io.Writer, command, args string) error {
	switch command {
	case "student-add":
		return showSaved(out, c.addStudent(args))
	case "criterion-add":
		return showSaved(out, c.addCriterion(args))
	case "grade-set", "grade-edit":
		return showSaved(out, c.setGrade(command, args))
	case "grade-cancel":
		fields := strings.Fields(args)
		if len(fields) != 2 {
			return fmt.Errorf("формат: grade-cancel СТУДЕНТ КРИТЕРИЙ")
		}
		return showSaved(out, c.gradebook.Cancel(fields[0], fields[1]))
	case "students", "criteria", "help", "exit":
		if args != "" {
			return fmt.Errorf("команда %s не принимает аргументы", command)
		}
		c.showList(out, command)
	case "grades":
		return c.showGrades(out, args)
	case "total":
		return c.showTotals(out, args)
	case "export":
		if err := c.gradebook.ExportTotals(args); err != nil {
			return err
		}
		fmt.Fprintln(out, "Итоги экспортированы:", args)
	default:
		return fmt.Errorf("неизвестная команда %q. Список команд: help", command)
	}
	return nil
}

func (c *Console) addStudent(args string) error {
	fields := splitLine(args, 2)
	if len(fields) != 2 {
		return fmt.Errorf("формат: student-add ID ИМЯ")
	}
	return c.gradebook.AddStudent(fields[0], fields[1])
}

func (c *Console) addCriterion(args string) error {
	fields := splitLine(args, 4)
	if len(fields) != 4 {
		return fmt.Errorf("формат: criterion-add ID МАКСИМУМ ВЕС НАЗВАНИЕ")
	}
	maxScore, err := number(fields[1], "максимальный балл")
	if err != nil {
		return err
	}
	weight, err := number(fields[2], "вес")
	if err != nil {
		return err
	}
	return c.gradebook.AddCriterion(fields[0], fields[3], maxScore, weight)
}

func (c *Console) setGrade(command, args string) error {
	fields := strings.Fields(args)
	if len(fields) != 3 {
		return fmt.Errorf("формат: %s СТУДЕНТ КРИТЕРИЙ БАЛЛ", command)
	}
	points, err := number(fields[2], "балл")
	if err != nil {
		return err
	}
	if command == "grade-edit" {
		return c.gradebook.Correct(fields[0], fields[1], points)
	}
	return c.gradebook.Assign(fields[0], fields[1], points)
}

func (c *Console) showList(out io.Writer, command string) {
	switch command {
	case "students":
		students := c.gradebook.Students()
		if len(students) == 0 {
			fmt.Fprintln(out, "Студентов пока нет.")
		}
		for _, student := range students {
			fmt.Fprintf(out, "%s\t%s\n", student.ID(), student.Name())
		}
	case "criteria":
		criteria := c.gradebook.Criteria()
		if len(criteria) == 0 {
			fmt.Fprintln(out, "Рубрикатор пуст.")
		}
		for _, criterion := range criteria {
			fmt.Fprintf(out, "%s\t%s\tмаксимум %g\tвес %g\n", criterion.ID(), criterion.Name(), criterion.MaxScore(), criterion.Weight())
		}
	case "help":
		fmt.Fprintln(out, help)
	}
}

func (c *Console) showGrades(out io.Writer, args string) error {
	fields := strings.Fields(args)
	if len(fields) != 1 {
		return fmt.Errorf("формат: grades СТУДЕНТ")
	}
	evaluations, err := c.gradebook.Evaluations(fields[0])
	if err != nil {
		return err
	}
	if len(evaluations) == 0 {
		fmt.Fprintln(out, "Оценок пока нет.")
	}
	for _, evaluation := range evaluations {
		fmt.Fprintf(out, "%s\t%g\n", evaluation.CriterionID(), evaluation.Score().Points())
	}
	return nil
}

func (c *Console) showTotals(out io.Writer, args string) error {
	if len(strings.Fields(args)) > 1 {
		return fmt.Errorf("формат: total [СТУДЕНТ]")
	}
	totals, err := c.gradebook.Totals(args)
	if err != nil {
		return err
	}
	if len(totals) == 0 {
		fmt.Fprintln(out, "Студентов пока нет.")
	}
	for _, row := range totals {
		result := "итог не определён"
		if row.Available {
			result = fmt.Sprintf("%.2f%%", row.Percent)
		}
		fmt.Fprintf(out, "%s\t%s\t%s\tоценено %d/%d\n", row.Student.ID(), row.Student.Name(), result, row.Graded, row.Criteria)
	}
	return nil
}

func showSaved(out io.Writer, err error) error {
	if err == nil {
		fmt.Fprintln(out, "Готово.")
	}
	return err
}

func number(raw, label string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: не удалось прочитать число %q, используйте десятичную точку", label, raw)
	}
	return value, nil
}

func splitLine(line string, limit int) []string {
	line = strings.TrimSpace(line)
	var fields []string
	for line != "" && len(fields) < limit-1 {
		i := strings.IndexFunc(line, unicode.IsSpace)
		if i < 0 {
			break
		}
		fields = append(fields, line[:i])
		line = strings.TrimSpace(line[i:])
	}
	if line != "" {
		fields = append(fields, line)
	}
	return fields
}
