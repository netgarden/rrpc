package rrpc_test

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc-test/modules/auth"
	authImpl "github.com/netgarden/rrpc-test/modules/auth/impl"
	"github.com/netgarden/rrpc-test/modules/echo"
	echoImpl "github.com/netgarden/rrpc-test/modules/echo/impl"
	"github.com/netgarden/rrpc-test/modules/files"
	filesImpl "github.com/netgarden/rrpc-test/modules/files/impl"
	"github.com/netgarden/rrpc-test/modules/test"
	testImpl "github.com/netgarden/rrpc-test/modules/test/impl"
)

func NewServer() *Server {
	return &Server{}
}

type Server struct {
	rrpcServer *rrpc.Server
	httpServer *http.Server
}

func (s *Server) Init() error {

	var err error

	authModule := auth.NewModule()

	authService := authImpl.NewAuthService()
	authModule.SetAuthService(authService)

	testModule := test.NewModule()

	pingerService := testImpl.NewPingerService()
	testModule.SetPingerService(pingerService)

	staticService := testImpl.NewStaticService("web/build")
	testModule.SetStaticService(staticService)

	filesModule := files.NewModule()
	filesService := filesImpl.NewFilesService("uploads")
	filesModule.SetFilesService(filesService)

	echoModule := echo.NewModule()
	echoModule.SetEchoService(echoImpl.NewEchoService())

	rrpcServer := rrpc.NewServer()

	err = rrpcServer.RegisterModule(authModule)
	if err != nil {
		return err
	}

	err = rrpcServer.RegisterModule(testModule)
	if err != nil {
		return err
	}

	err = rrpcServer.RegisterModule(filesModule)
	if err != nil {
		return err
	}

	err = rrpcServer.RegisterModule(echoModule)
	if err != nil {
		return err
	}

	s.rrpcServer = rrpcServer

	return nil
}

func (s *Server) Start() error {

	s.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(s.rrpcServer.ServeHTTP),
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
		log.Println("Stopped serving new connections")
	}()

	return nil
}

func (s *Server) Stop() {

	log.Println("Shutting down gracefully...")

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
	}

	log.Println("Server stopped!")
}
