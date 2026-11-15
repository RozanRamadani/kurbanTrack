package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/app/service"
	"kurbantrack-backend/config"
	"kurbantrack-backend/database"
)

func main() {
	// 1. Memuat konfigurasi environment
	cfg := config.LoadConfig()

	// 2. Inisialisasi logger
	appLogger := config.InitLogger()
	appLogger.Info.Println("Inisialisasi KurbanTrack Backend...")

	// 3. Menghubungkan koneksi pool PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := database.NewPostgresPool(ctx, cfg.GetDSN())
	if err != nil {
		appLogger.Error.Printf("Peringatan: Gagal membuat pool database: %v", err)
	} else {
		defer dbPool.Close()
		if err := dbPool.Ping(ctx); err != nil {
			appLogger.Error.Printf("Peringatan: Database belum dapat dihubungi: %v", err)
		} else {
			appLogger.Info.Println("Koneksi ke PostgreSQL berhasil terhubung.")
		}
	}

	// 4. Inisialisasi repositories
	animalRepo := repository.NewAnimalRepository(dbPool)
	distributionRepo := repository.NewDistributionRepository(dbPool)

	// 5. Inisialisasi services
	healthService := service.NewHealthService(dbPool)
	animalService := service.NewAnimalService(animalRepo)
	distributionService := service.NewDistributionService(distributionRepo, animalRepo)

	// 6. Inisialisasi aplikasi Fiber dengan dependency services
	app := config.NewApp(healthService, animalService, distributionService)

	// 7. Menangani graceful shutdown dengan batas waktu 5 detik
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		appLogger.Info.Println("Menerima sinyal terminasi, mematikan server secara graceful (timeout 5 detik)...")
		if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
			appLogger.Error.Printf("Error saat shutdown server: %v", err)
		}
	}()

	// 8. Menjalankan server
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	appLogger.Info.Printf("Server berjalan pada port %s", cfg.AppPort)
	if err := app.Listen(addr); err != nil {
		appLogger.Info.Printf("Server berhenti: %v", err)
	}
}
