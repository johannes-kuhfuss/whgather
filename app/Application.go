package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/services_utils/date"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/whgather/config"
)

var (
	cfg    config.AppConfig
	server http.Server
	appEnd chan os.Signal
	ctx    context.Context
	cancel context.CancelFunc
)

func StartApp() {
	logger.Info("Starting application")

	err := config.InitConfig(&cfg)
	if err != nil {
		panic(err)
	}
	initRouter()
	initServer()
	wireApp()
	mapUrls()
	RegisterForOsSignals()
	go startServer()

	<-appEnd
	cleanUp()

	if srvErr := server.Shutdown(ctx); srvErr != nil {
		logger.Error("Graceful shutdown failed", srvErr)
	} else {
		logger.Info("Graceful shutdown finished")
	}
}

func initRouter() {
	gin.SetMode(cfg.Gin.Mode)
	gin.DefaultWriter = logger.GetLogger()
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.SetTrustedProxies(nil)

	cfg.RunTime.Router = router
}

func initServer() {
	var tlsConfig tls.Config

	if cfg.Server.UseTls {
		tlsConfig = tls.Config{
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		}
	}
	if cfg.Server.UseTls {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.TlsPort)
	} else {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	}

	server = http.Server{
		Addr:              cfg.RunTime.ListenAddr,
		Handler:           cfg.RunTime.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 0,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    0,
	}
	if cfg.Server.UseTls {
		server.TLSConfig = &tlsConfig
		server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

func wireApp() {
}

func mapUrls() {
}

func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func startServer() {
	logger.Info(fmt.Sprintf("Listening on %v", cfg.RunTime.ListenAddr))
	cfg.RunTime.StartDate = date.GetNowUtc()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Error while starting http server", err)
		panic(err)
	}
}

func cleanUp() {
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Cleaning up")
		logger.Info("Done cleaning up")
		cancel()
	}()
}
