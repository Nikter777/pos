package csvexport

import (
	"errors"
	"testing"
)

type failedOutput struct{ err error }

func (w failedOutput) Write([]byte) (int, error) { return 0, w.err }

func TestWriteTotalsPropagatesFlushError(t *testing.T) {
	expected := errors.New("ошибка записи")
	if err := writeTotals(failedOutput{err: expected}, nil); !errors.Is(err, expected) {
		t.Fatalf("ошибка записи потеряна: %v", err)
	}
}
