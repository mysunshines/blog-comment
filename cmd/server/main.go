package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mysunshines/blog-comment/internal/config"
	"github.com/mysunshines/blog-comment/internal/handler"
	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/blog-comment/internal/repository"
	"github.com/mysunshines/blog-comment/internal/service"
	comment "github.com/mysunshines/blog-comment/proto/pb"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/cache"
	"github.com/mysunshines/gocommon/constants"
	common_database "github.com/mysunshines/gocommon/database"
	"github.com/mysunshines/gocommon/log"
	"github.com/mysunshines/gocommon/metrics"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

// Server 服务器结构
type Server struct {
	cfg             *config.Config
	httpServer      *http.Server
	grpcServer      *grpc.Server
	commentSvc      service.CommentService
	commentRepo     repository.CommentRepository
	commentLikeRepo repository.CommentLikeRepository
	commentHandl    *handler.CommentHandler
	db              *gorm.DB
	userClient      user.UserServiceClient
	cb              *gobreaker.CircuitBreaker // 熔断器
}

// NewServer 创建服务器
func NewServer(cfg *config.Config) *Server {
	// 初始化数据库（类型别名，直接传递）
	if err := common_database.Init(&cfg.Database, cfg.App.Env); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	db := common_database.GetDB()

	// 初始化 Redis 缓存（必须在 AutoMigrate 之前，用于分布式锁）
	redisCfg := cfg.Redis
	redisCfg.KeyPrefix = constants.RedisKeyPrefixComment
	if redisCfg.PoolSize == 0 {
		redisCfg.PoolSize = 100
	}
	if err := cache.Init(&redisCfg); err != nil {
		log.Warnf("Warning: Failed to init Redis: %v", err)
	}

	// 自动迁移（分布式锁保护，多实例只有一个执行）
	const migrationLockKey = "migration:lock:comment_service"
	const migrationLockTTL = 60 * time.Second
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	acquired, err := cache.TryLock(context.Background(), migrationLockKey, instanceID, migrationLockTTL)
	if err != nil {
		log.Warnf("Failed to acquire migration lock (Redis unavailable): %v, proceeding without lock", err)
	} else if acquired {
		log.Infof("Migration lock acquired by instance %s", instanceID)
		defer func() {
			if unlockErr := cache.Unlock(context.Background(), migrationLockKey, instanceID); unlockErr != nil {
				log.Warnf("Failed to release migration lock: %v", unlockErr)
			}
		}()
	} else {
		log.Info("Migration lock held by another instance, skipping AutoMigrate")
		time.Sleep(2 * time.Second)
	}

	if acquired || err != nil {
		if migrateErr := db.AutoMigrate(&model.Comment{}, &model.CommentLike{}, &model.Article{}, &model.User{}); migrateErr != nil {
			log.Fatalf("Failed to migrate database: %v", migrateErr)
		}
	}

	// 初始化限流器（类型别名，直接传递）
	commonmiddleware.InitRateLimiter(&cfg.RateLimit)

	// 初始化 JWT
	commonmiddleware.InitJWT(cfg.JWT.Secret)

	// 初始化熔断器
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        constants.ServiceNameComment,
		MaxRequests: constants.DefaultCBMaxRequests,
		Interval:    constants.DefaultCBInterval * time.Second,
		Timeout:     constants.DefaultCBTimeout * time.Second,
	})

	// 连接 User Service（带超时和重试）
	userClient, err := initUserClient(cfg)
	if err != nil {
		log.Warnf("Warning: Failed to connect to user service: %v", err)
	}

	// 初始化仓储层
	commentRepo := repository.NewCommentRepository(db)
	commentLikeRepo := repository.NewCommentLikeRepository(db)

	// 初始化服务层
	commentSvc := service.NewCommentService(commentRepo, commentLikeRepo, db, userClient)

	// 初始化处理器
	commentHandl := handler.NewCommentHandler(commentSvc)

	return &Server{
		cfg:             cfg,
		commentSvc:      commentSvc,
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		commentHandl:    commentHandl,
		db:              db,
		userClient:      userClient,
		cb:              cb,
	}
}

func initUserClient(cfg *config.Config) (user.UserServiceClient, error) {
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 使用默认服务配置，启用自动重连和 waitForReady
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy": "round_robin",
			"healthCheckConfig": {
				"serviceName": ""
			}
		}`),
		// Keepalive 参数：定期探测连接是否存活，断线自动重连
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // 10s 发送一次心跳
			Timeout:             3 * time.Second,  // 心跳超时 3s
			PermitWithoutStream: true,             // 无活动流也发心跳
		}),
	}

	conn, err := grpc.NewClient(cfg.UserService.Addr(), grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service client: %w", err)
	}
	return user.NewUserServiceClient(conn), nil
}

// Run 运行服务器
func (s *Server) Run() error {
	// 启动 HTTP 服务器
	go s.runHTTPServer()

	// 启动 gRPC 服务器
	go s.runGRPCServer()

	// 启动 Prometheus 指标服务器
	if s.cfg.Metrics.Enabled {
		go s.runMetricsServer()
	}

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Errorf("HTTP server shutdown error: %v", err)
	}

	s.grpcServer.GracefulStop()

	log.Info("Server exited")
	return nil
}

// runHTTPServer 运行 HTTP 服务器
func (s *Server) runHTTPServer() {
	if s.cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(commonmiddleware.RecoveryMiddleware())
	router.Use(commonmiddleware.LoggingMiddleware())
	router.Use(commonmiddleware.CORSMiddleware())
	router.Use(commonmiddleware.MetricsMiddleware(constants.ServiceNameComment))
	router.Use(commonmiddleware.TraceMiddleware())

	// 高并发增强：请求超时中间件
	router.Use(commonmiddleware.TimeoutMiddleware(30 * time.Second))

	// 高并发增强：限流中间件
	if s.cfg.RateLimit.Enabled {
		router.Use(commonmiddleware.RateLimitMiddleware())
	}

	// 健康检查（带深度检查）
	router.GET("/health", func(c *gin.Context) {
		// 检查数据库连接
		if sqlDB, _ := s.db.DB(); sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "db"})
				return
			}
		}

		// 检查 Redis 连接
		if err := cache.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "redis"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 就绪探针
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API 路由
	api := router.Group(constants.APIPathPrefix)
	{
		commentGroup := api.Group("/comment")
		{
			// 公开接口
			commentGroup.GET("", s.commentHandl.ListComments)
			commentGroup.GET("/:id", s.commentHandl.GetComment)
			commentGroup.GET("/article/:article_id", s.commentHandl.GetArticleComments)
			commentGroup.GET("/:id/replies", s.commentHandl.GetCommentReplies)

			// 需要登录的接口
			commentGroup.POST("", commonmiddleware.JWTValidMiddleware(), s.commentHandl.CreateComment)
			commentGroup.PUT("/:id", commonmiddleware.JWTValidMiddleware(), s.commentHandl.UpdateComment)
			commentGroup.DELETE("/:id", commonmiddleware.JWTValidMiddleware(), s.commentHandl.DeleteComment)
			commentGroup.POST("/:id/reply", commonmiddleware.JWTValidMiddleware(), s.commentHandl.ReplyComment)
			commentGroup.POST("/:id/like", commonmiddleware.JWTValidMiddleware(), s.commentHandl.LikeComment)
			commentGroup.POST("/article/:article_id/enable", commonmiddleware.JWTValidMiddleware(), s.commentHandl.EnableComment)
			commentGroup.POST("/article/:article_id/disable", commonmiddleware.JWTValidMiddleware(), s.commentHandl.DisableComment)
		}
	}

	addr := s.cfg.HTTP.Addr()

	// 高并发增强：配置 HTTP Server 超时
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       constants.DefaultReadTimeout * time.Second,
		ReadHeaderTimeout: constants.DefaultReadHeaderTimeout * time.Second,
		WriteTimeout:      constants.DefaultWriteTimeout * time.Second,
		IdleTimeout:       constants.DefaultIdleTimeout * time.Second,
		MaxHeaderBytes:    constants.MaxHeaderBytes,
	}

	log.Infof("HTTP server starting on %s (timeouts: read=%v, write=%v, idle=%v)", addr,
		s.httpServer.ReadTimeout, s.httpServer.WriteTimeout, s.httpServer.IdleTimeout)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

// runGRPCServer 运行 gRPC 服务器
func (s *Server) runGRPCServer() {
	lis, err := net.Listen("tcp", s.cfg.GRPC.Addr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 高并发增强：gRPC 选项配置
	grpcOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     constants.DefaultGRPCMaxConnectionIdle * time.Second,
			MaxConnectionAge:      constants.DefaultGRPCMaxConnectionAge * time.Second,
			MaxConnectionAgeGrace: constants.DefaultGRPCMaxConnectionAgeGrace * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             constants.DefaultGRPCMinPingInterval * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxConcurrentStreams(constants.DefaultGRPCMaxConcurrentStreams),
	}

	// 高并发增强：添加 unary 拦截器（超时+熔断）
	grpcOpts = append(grpcOpts, grpc.UnaryInterceptor(s.grpcUnaryInterceptor))

	s.grpcServer = grpc.NewServer(grpcOpts...)
	comment.RegisterCommentServiceServer(s.grpcServer, &handler.GrpcCommentHandler{
		Svc: s.commentSvc,
		Cb:  s.cb,
	})
	reflection.Register(s.grpcServer)

	log.Infof("gRPC server starting on %s", s.cfg.GRPC.Addr())
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

// grpcUnaryInterceptor gRPC 一元拦截器（超时+熔断）
func (s *Server) grpcUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// 超时控制
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultGRPCUnaryTimeout*time.Second)
	defer cancel()

	// 熔断器保护
	if s.cb != nil {
		result, err := s.cb.Execute(func() (interface{}, error) {
			return handler(ctx, req)
		})
		return result, err
	}

	return handler(ctx, req)
}

// runMetricsServer 运行指标服务器
func (s *Server) runMetricsServer() {
	addr := fmt.Sprintf(":%d", s.cfg.Metrics.Port)
	http.Handle(s.cfg.Metrics.Path, promhttp.Handler())

	log.Infof("Metrics server starting on %s%s", addr, s.cfg.Metrics.Path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Errorf("Metrics server error: %v", err)
	}
}

func main() {
	// 加载配置（支持 CONFIG_PATH 环境变量覆盖，与 user/article 服务保持一致）
	configPath := os.Getenv(constants.EnvConfigPath)
	if configPath == "" {
		configPath = constants.DefaultConfigPath
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	log.Init(cfg.App.LogDir, cfg.App.LogLevel, constants.ServiceNameComment)

	// 初始化指标
	metrics.Init()

	// 创建并运行服务器
	server := NewServer(cfg)
	defer common_database.Close()
	defer cache.Close()
	defer log.StopRotation()
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
