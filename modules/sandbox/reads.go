package sandbox

func (r *Registry) Environment(id string) (Environment, error) {
	r.enter()
	defer r.leave()
	env := r.environments[id]
	if env == nil {
		return Environment{}, ErrNotFound
	}
	return env.view, nil
}

func (r *Registry) Destinations(environment string) ([]DestinationView, error) {
	r.enter()
	defer r.leave()
	env := r.environments[environment]
	if env == nil {
		return nil, ErrNotFound
	}
	out := []DestinationView{}
	for _, id := range env.destinations {
		d := r.destinations[id]
		out = append(out, DestinationView{Destination: d.view, Enabled: d.enabled})
	}
	return out, nil
}

func (r *Registry) Schema(id string) (Schema, error) {
	r.enter()
	defer r.leave()
	s := r.schemas[id]
	if s == nil {
		return Schema{}, ErrNotFound
	}
	return copySchema(*s), nil
}
