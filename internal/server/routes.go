package server

import (
	"net/http"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/core/domain"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()
	if s.httpLog {
		e.Use(middleware.Logger())
	}
	e.Use(middleware.Recover())

	e.GET("/healthcheck", s.HealthCheckHandler)

	return e
}

func (s *Server) HealthCheckHandler(c echo.Context) error {
	res, err := s.rootContext.RequestFuture(s.masterActor, domain.ActorHealthRequest{}, 10*time.Second).Result()
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "health_check: FAIL")
	}
	if response, ok := res.(domain.ActorHealthResponse); ok && response.Healthy {
		return c.String(http.StatusOK, "health_check: OK")
	}
	return c.String(http.StatusServiceUnavailable, "health_check: FAIL")
}
