package server

import (
	"fmt"
	"frostnews2mqtt/internal/config"
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port        uint
	rootContext *actor.RootContext
	masterActor *actor.PID
}

func NewServer(cfg config.Config, rootContext *actor.RootContext, masterActor *actor.PID) *http.Server {
	NewServer := &Server{
		port:        cfg.Port,
		rootContext: rootContext,
		masterActor: masterActor,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
