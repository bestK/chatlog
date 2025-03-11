package http

import (
	"context"
	"net/http"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/sjzar/chatlog/internal/chatlog/mcp"
	"github.com/sjzar/chatlog/internal/errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ctx *ctx.Context
	db  *database.Service
	mcp *mcp.Service

	router *gin.Engine
	server *http.Server
}

func NewService(ctx *ctx.Context, db *database.Service, mcp *mcp.Service) *Service {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Handle error from SetTrustedProxies
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Error("Failed to set trusted proxies:", err)
	}

	// Middleware
	router.Use(
		gin.Recovery(),
		gin.LoggerWithWriter(log.StandardLogger().Out),
	)

	s := &Service{
		ctx:    ctx,
		db:     db,
		mcp:    mcp,
		router: router,
	}

	s.initRouter()
	return s
}

func (s *Service) Start() error {
	s.server = &http.Server{
		Addr:    s.ctx.HTTPAddr,
		Handler: s.router,
	}

	go func() {
		// Handle error from Run
		if err := s.server.ListenAndServe(); err != nil {
			log.Error("Server Stopped: ", err)
		}
	}()

	log.Info("Server started on ", s.ctx.HTTPAddr)

	return nil
}

func (s *Service) Stop() error {

	if s.server == nil {
		return nil
	}

	// 使用超时上下文优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return errors.HTTP("HTTP server shutdown error", err)
	}

	log.Info("HTTP server stopped")
	return nil
}

func (s *Service) GetRouter() *gin.Engine {
	return s.router
}
