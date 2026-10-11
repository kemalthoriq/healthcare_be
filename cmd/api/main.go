package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/kemalthoriq/healthcare_be/db/migrations"
	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/invoice"
	"github.com/kemalthoriq/healthcare_be/internal/notify"
	"github.com/kemalthoriq/healthcare_be/internal/platform/captcha"
	"github.com/kemalthoriq/healthcare_be/internal/platform/config"
	"github.com/kemalthoriq/healthcare_be/internal/platform/database"
	"github.com/kemalthoriq/healthcare_be/internal/platform/email"
	"github.com/kemalthoriq/healthcare_be/internal/platform/logger"
	"github.com/kemalthoriq/healthcare_be/internal/platform/ratelimit"
	"github.com/kemalthoriq/healthcare_be/internal/promo"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.IsProduction())
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	log.Info("database connected", "max_conns", cfg.DBMaxConns)

	if err := database.Migrate(ctx, pool, migrations.FS, log); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	authService := auth.NewService(auth.NewRepository(pool), log, cfg.SessionTTL)
	if err := authService.EnsureAdmin(ctx, auth.AdminSeed{
		Name:     cfg.SeedAdminName,
		Email:    cfg.SeedAdminEmail,
		Password: cfg.SeedAdminPassword,
	}); err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}

	clinicLocation, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return fmt.Errorf("load clinic timezone: %w", err)
	}

	bookingRepository := booking.NewRepository(pool)
	treatmentService := treatment.NewService(treatment.NewRepository(pool), bookingRepository)
	therapistService := therapist.NewService(therapist.NewRepository(pool), treatmentService, bookingRepository)
	accountService := therapist.NewAccountService(therapistService, authService)
	settingService := setting.NewService(setting.NewRepository(pool))
	availability := booking.NewAvailability(treatmentService, therapistService, settingService, bookingRepository, clinicLocation)
	var sender email.Sender
	if cfg.EmailEnabled {
		smtpSender, err := email.NewSMTPSender(email.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		})
		if err != nil {
			return fmt.Errorf("create email sender: %w", err)
		}
		sender = smtpSender
	}
	mailer := email.NewQueue(sender, log)
	notifier := notify.New(mailer, settingService, clinicLocation, cfg.FrontendOrigin, log)
	bookingService := booking.NewService(availability, bookingRepository, notifier)
	invoiceService := invoice.NewService(invoice.NewRepository(pool), bookingService, clinicLocation)

	verifier := captcha.New(cfg.TurnstileSecret, log)
	if !verifier.Enabled() {
		log.Warn("TURNSTILE_SECRET_KEY not set, captcha verification disabled")
	}

	router := newRouter(routerDeps{
		log:              log,
		db:               pool,
		frontendOrigin:   cfg.FrontendOrigin,
		trustedProxyHops: cfg.TrustedProxyHops,
		limits:           newRateLimits(),
		auth:             auth.NewHandler(authService, cfg.IsProduction(), ratelimit.New(10, 15*time.Minute)),
		treatment:        treatment.NewHandler(treatmentService),
		therapist:        therapist.NewHandler(therapistService),
		account:          therapist.NewAccountHandler(accountService),
		setting:          setting.NewHandler(settingService),
		booking:          booking.NewHandler(availability, bookingService, verifier, clinicLocation),
		patient:          booking.NewPatientHandler(booking.NewPatientService(bookingRepository, bookingService), clinicLocation),
		invoice:          invoice.NewHandler(invoiceService, clinicLocation),
		promo:            promo.NewHandler(promo.NewService(promo.NewRepository(pool), clinicLocation)),
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server started", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		stop()
	}

	log.Info("shutting down server", "timeout", cfg.ShutdownTimeout)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	if err := mailer.Close(shutdownCtx); err != nil {
		log.Warn("email queue not fully sent before shutdown", "err", err)
	}

	log.Info("server stopped")
	return nil
}
