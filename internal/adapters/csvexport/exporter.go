package csvexport

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"pos/internal/domain"
)

type Exporter struct{}

func (Exporter) Export(path string, rows []domain.Total) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("укажите путь к CSV-файлу")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("создание CSV: %w", err)
	}

	writeErr := writeTotals(file, rows)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		if writeErr == nil {
			writeErr = closeErr
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("запись CSV: %w, удаление неполного файла: %w", writeErr, err)
		}
		return fmt.Errorf("запись CSV: %w", writeErr)
	}
	return nil
}

func writeTotals(output io.Writer, rows []domain.Total) error {
	writer := csv.NewWriter(output)
	if err := writer.Write([]string{"student_id", "student_name", "total_percent", "graded", "criteria"}); err != nil {
		return err
	}
	for _, row := range rows {
		percent := ""
		if row.Available {
			percent = strconv.FormatFloat(row.Percent, 'f', 2, 64)
		}
		err := writer.Write([]string{
			row.Student.ID(), row.Student.Name(), percent,
			strconv.Itoa(row.Graded), strconv.Itoa(row.Criteria),
		})
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
