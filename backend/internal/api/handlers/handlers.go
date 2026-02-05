package handlers

type Handlers struct {
	Auth       *AuthHandler
	Job        *JobHandler
	Customer   *CustomerHandler
	Technician *WorkerHandler
	Files      *FileHandler
	Health     *HealthHandler
}

// NewHandlers creates the handlers struct (just grouping, not wiring)
func NewHandlers(
	auth *AuthHandler,
	job *JobHandler,
	customer *CustomerHandler,
	technician *WorkerHandler,
	files *FileHandler,
	health *HealthHandler,
) *Handlers {
	return &Handlers{
		Auth:       auth,
		Job:        job,
		Customer:   customer,
		Technician: technician,
		Files:      files,
		Health:     health,
	}
}
