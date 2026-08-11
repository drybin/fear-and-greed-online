package strategy

import "fmt"

type Registry struct {
	items map[string]Strategy
}

func NewRegistry(strategies ...Strategy) *Registry {
	items := make(map[string]Strategy, len(strategies))
	for _, s := range strategies {
		items[s.Slug()] = s
	}
	return &Registry{items: items}
}

func (r *Registry) Get(slug string) (Strategy, error) {
	item, ok := r.items[slug]
	if !ok {
		return nil, fmt.Errorf("strategy not registered: %s", slug)
	}
	return item, nil
}
