package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	primaryhttp "github.com/example/starter-service/internal/adapters/primary/http"
	"github.com/example/starter-service/internal/adapters/secondary/memory"
	"github.com/example/starter-service/internal/platform/config"
	"github.com/example/starter-service/internal/usecases"
)

func main() {
	// 1. إعداد مسجل السجلات المهيكل (Structured Logger)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("starting starter-service...")

	// 2. تحميل الإعدادات والتحقق الصارم منها (Fail-Fast)
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration loading failed", "error", err)
		os.Exit(1)
	}

	// 3. إنشاء سياق الإيقاف المبكر المرتبط بإشارات نظام التشغيل
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. جذر التكوين (Composition Root): حقن التبعيات بالترتيب من الداخل للخارج
	// محول التخزين الثانوي (Secondary Adapter)
	walletRepo := memory.NewInMemWalletRepository()

	// حالة الاستخدام (Use Case / Interactor) - تقبل الواجهة وتُرجع هيكلاً ملموساً
	transferUC := usecases.NewTransferFundsUseCase(walletRepo, walletRepo)

	// محول واجهة الـ HTTP الأولي (Primary Adapter)
	walletHandler := primaryhttp.NewWalletHandler(transferUC)

	// إعداد خادم الويب
	srv := primaryhttp.NewServer(cfg, walletHandler, logger)

	// 5. تشغيل الخادم في Goroutine منفصلة
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start()
	}()

	logger.Info("application initialized successfully", "port", cfg.Port, "env", cfg.Environment)

	// 6. مراقبة إشارات الإيقاف أو أخطاء الخادم غير المتوقعة
	select {
	case err := <-serverErrors:
		logger.Error("server encountered fatal startup error", "error", err)
		os.Exit(1)

	case <-ctx.Done():
		logger.Info("shutdown signal received; commencing graceful teardown...")

		// تحديد مهلة زمنية قاطعة لإنهاء الطلبات العالقة
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed; forcing termination", "error", err)
			os.Exit(1)
		}

		logger.Info("service exited cleanly")
	}
}
