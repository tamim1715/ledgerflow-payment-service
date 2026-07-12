package handler

type Handlers struct {
	Health *HealthHandler
}

func NewHandler(health *HealthHandler) *Handlers {
	return &Handlers{
		Health: health,
	}
}
