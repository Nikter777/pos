package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"pos/internal/composition"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, input io.Reader, output, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("pos", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	missing := flags.String("missing", "zero", "учёт пропущенных оценок: zero или ignore")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, "Неожиданные аргументы. Справка: pos -help")
		return 2
	}
	container, err := composition.New(*missing)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 2
	}
	if err := container.Console.Run(input, output); err != nil {
		fmt.Fprintln(diagnostics, "Ошибка консоли:", err)
		return 1
	}
	return 0
}
