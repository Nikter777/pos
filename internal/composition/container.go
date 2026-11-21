package composition

import (
	"fmt"

	"pos/internal/adapters/console"
	"pos/internal/adapters/csvexport"
	"pos/internal/adapters/memory"
	"pos/internal/application"
	"pos/internal/domain"
)

type Container struct {
	Console *console.Console
}

func New(missing string) (*Container, error) {
	var policy domain.MissingPolicy
	switch missing {
	case "zero":
		policy = domain.MissingAsZero{}
	case "ignore":
		policy = domain.IgnoreMissing{}
	default:
		return nil, fmt.Errorf("неизвестная политика %q, выберите zero или ignore", missing)
	}
	repo := memory.NewRepository()
	calculator := domain.NewCalculator(policy)
	app := application.NewGradebook(repo, calculator, csvexport.Exporter{})
	return &Container{Console: console.New(app)}, nil
}
