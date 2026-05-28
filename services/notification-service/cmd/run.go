package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auron/notification-service/internal/email"
	"auron/notification-service/internal/events"
	"auron/notification-service/internal/handler"
	"auron/notification-service/internal/service"
)

func Run() {
	cfg := loadConfig()

	sender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPSecure)

	notificationSvc := service.NewNotificationService(sender)

	ctx, cancel := context.WithCancel(context.Background())
	consumer := setupKafkaConsumer(cfg.KafkaBrokers, notificationSvc)
	startKafkaConsumer(ctx, consumer)

	healthHandler := handler.NewHealthHandler()
	router := setupRouter(healthHandler)

	registerGracefulShutdown(consumer, cancel)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting notification-service on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerGracefulShutdown(consumer *events.KafkaConsumer, cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down notification-service...")
		cancel()
		closeKafkaConsumer(consumer)
		os.Exit(0)
	}()
}
