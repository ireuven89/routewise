package handlers

type Handlers struct {
	Auth       *AuthHandler
	Job        *JobHandler
	Customer   *CustomerHandler
	Technician *WorkerHandler
	Files      *FileHandler
	Health     *HealthHandler
	Geocoding  *GeocodingHandler
}

// NewHandlers creates the handlers struct (just grouping, not wiring)
func NewHandlers(
	auth *AuthHandler,
	job *JobHandler,
	customer *CustomerHandler,
	technician *WorkerHandler,
	files *FileHandler,
	health *HealthHandler,
	geocoding *GeocodingHandler,
) *Handlers {
	return &Handlers{
		Auth:       auth,
		Job:        job,
		Customer:   customer,
		Technician: technician,
		Files:      files,
		Health:     health,
		Geocoding:  geocoding,
	}
}
