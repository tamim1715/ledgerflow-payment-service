package main

import (
	"log"

	"ledgerflow-payment-service/internal/app"
)

// @title           LedgerFlow Payment & Billing Service API
// @version         1.0
// @description     Enterprise Multi-Tenant Payment Gateway & Automated Billing Microservice
// @termsOfService  http://swagger.io/terms/

// @contact.name    LedgerFlow Engineering Team
// @contact.url     https://ledgerflow.io/support
// @contact.email   support@ledgerflow.io

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:8081
// @BasePath        /
// @schemes         http https
func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		application.Logger.Fatal(err.Error())
	}
}
