package main

import (
	"database/sql"
	"log"
	"net/http"
	"xm-company-service/internal/handler"
	"xm-company-service/internal/kafka"
	"xm-company-service/internal/middleware"
	"xm-company-service/internal/repository"
	"xm-company-service/internal/service"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq" // Postgres driver
)

func main() {
	// 1. Init DB
	db, err := sql.Open("postgres", "postgres://user:pass@db:5432/xm?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Init Kafka Producer (Plus requirement)
	// We would create a concrete implementation of EventProducer here
	producer := kafka.NewProducer([]string{"kafka:9092"})

	// 3. Dependency Injection
	repo := repository.NewPostgresRepo(db)
	svc := service.NewCompanyService(repo, producer)
	h := handler.NewHandler(svc)

	// 4. Routing
	r := chi.NewRouter()

	// Public Routes
	r.Get("/companies/{id}", h.GetCompany)

	// Protected Routes [cite: 2]
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth) // Auth required for mutation
		r.Post("/companies", h.CreateCompany)
		r.Patch("/companies/{id}", h.PatchCompany)
		r.Delete("/companies/{id}", h.DeleteCompany)
	})

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
