package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"r1rpc/internal/app"
	"r1rpc/internal/config"
	"r1rpc/internal/persistence"
	"r1rpc/internal/store"
	"r1rpc/internal/taskqueue"
	"r1rpc/internal/web"

	"github.com/gogf/gf/v2/frame/g"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		g.Log().Fatalf(context.Background(), "加载配置失败: %+v", err)
	}
	if err := cfg.ApplyTimeZone(); err != nil {
		g.Log().Fatalf(context.Background(), "应用时区失败: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := store.BootstrapSchema(ctx, cfg); err != nil {
		g.Log().Fatalf(context.Background(), "初始化数据库结构失败: %+v", err)
	}
	if err := persistence.ConfigureGoFrameDB(ctx, cfg); err != nil {
		g.Log().Fatalf(context.Background(), "初始化 GoFrame ORM 失败: %+v", err)
	}

	st, err := store.New(cfg)
	if err != nil {
		g.Log().Fatalf(context.Background(), "打开数据库失败: %+v", err)
	}

	application := app.New(cfg, st)
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭应用失败: %+v", closeErr)
		}
	}()
	if err := application.EnsureDeviceGroup(context.Background()); err != nil {
		g.Log().Fatalf(context.Background(), "初始化设备分组失败: %+v", err)
	}
	if err := application.Store.EnsureBootstrapAdmin(context.Background(), cfg.BootstrapAdminUser, cfg.BootstrapAdminPass); err != nil {
		g.Log().Fatalf(context.Background(), "初始化管理员失败: %+v", err)
	}
	rebuildCtx, rebuildCancel := context.WithTimeout(context.Background(), 20*time.Second)
	if err := application.Store.RebuildRecentMetricsFromRequests(rebuildCtx, cfg.RawRetentionDays); err != nil {
		g.Log().Warningf(context.Background(), "重建近期设备指标失败: %+v", err)
	}
	rebuildCancel()
	runCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	application.StartBackgroundJobs(runCtx)
	queueRuntime := taskqueue.NewRuntime(application,
		taskqueue.RedisOptions(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB),
		cfg.Redis.APIWorkerConcurrency,
		cfg.Redis.DownloadWorkerConcurrency,
		cfg.Redis.ScanInterval,
		cfg.Redis.DownloadInterval,
	)
	application.ImageTasks.SetEnqueuer(queueRuntime.Enqueuer())
	application.ImageSearch.SetEnqueuer(queueRuntime.Enqueuer())
	application.DataSources.SetEnqueuer(queueRuntime.Enqueuer())
	if err := queueRuntime.Start(runCtx); err != nil {
		g.Log().Fatalf(context.Background(), "启动图片任务队列失败: %+v", err)
	}
	defer func() {
		if closeErr := queueRuntime.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭图片任务队列失败: %+v", closeErr)
		}
	}()
	webServer := web.New(application)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           webServer.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	g.Log().Info(context.Background(), g.Map{
		"event":     "server_start",
		"address":   cfg.HTTPAddr,
		"time_zone": cfg.TimeZone,
	})
	g.Log().Info(context.Background(), g.Map{"event": "invoke_auth", "mode": "group-scoped"})
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.ListenAndServe()
	}()
	select {
	case <-runCtx.Done():
		if closeErr := queueRuntime.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭图片任务队列失败: %+v", closeErr)
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		shutdownErr := server.Shutdown(shutdownCtx)
		shutdownCancel()
		if shutdownErr != nil {
			g.Log().Errorf(context.Background(), "HTTP 优雅退出失败: %+v", shutdownErr)
			_ = server.Close()
		}
		wsShutdownCtx, wsShutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if wsErr := webServer.ShutdownClientConnections(wsShutdownCtx); wsErr != nil {
			g.Log().Errorf(context.Background(), "WebSocket 排空失败: %+v", wsErr)
		}
		wsShutdownCancel()
		if err := <-serveDone; err != nil && err != http.ErrServerClosed {
			g.Log().Errorf(context.Background(), "HTTP 服务异常退出: %+v", err)
		}
		if closeErr := application.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭应用失败: %+v", closeErr)
		}
	case err := <-serveDone:
		if closeErr := queueRuntime.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭图片任务队列失败: %+v", closeErr)
		}
		if err != nil && err != http.ErrServerClosed {
			g.Log().Errorf(context.Background(), "HTTP 服务异常退出: %+v", err)
		}
		wsShutdownCtx, wsShutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if wsErr := webServer.ShutdownClientConnections(wsShutdownCtx); wsErr != nil {
			g.Log().Errorf(context.Background(), "WebSocket 排空失败: %+v", wsErr)
		}
		wsShutdownCancel()
		if closeErr := application.Close(); closeErr != nil {
			g.Log().Errorf(context.Background(), "关闭应用失败: %+v", closeErr)
		}
	}
}
