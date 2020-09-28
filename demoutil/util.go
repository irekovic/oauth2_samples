package demoutil

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/oklog/run"
)

func RunHTTPServer(bind string) (func() error, func(error)) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err.Error())
	}
	return func() error { return http.Serve(l, nil) }, func(err error) { l.Close() }
}

func RunSignalHandler() (func() error, func(error)) {
	return run.SignalHandler(context.Background(), os.Interrupt, os.Kill)
}
