package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/server"
)

func main() {
	fDebug := flag.Bool("debug", false, "enable debug logs")
	fAccessLog := flag.Bool("logaccess", true, "enables access logs")
	fHost := flag.String("host", "localhost:8080", "server host address")
	flag.Parse()

	var slogHandler slog.Handler
	if *fDebug {
		slogHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		slogHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(slogHandler)

	slog.SetDefault(logger)
	slog.Debug("debug mode enabled")

	store := domain.New()

	writeMockData(store)

	srv := server.New(store, *fAccessLog)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	s := &http.Server{
		Addr:        *fHost,
		Handler:     srv,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		slog.Info("listening", slog.String("host", *fHost))
		if err := s.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("serving http", slog.Any("err", err))
			}
		}
	})

	<-ctx.Done() // Wait until shutdown signal is received.
	if err := s.Shutdown(context.Background()); err != nil {
		slog.Error("shutting down HTTP server", slog.Any("err", err))
	}

	wg.Wait()
}

func writeMockData(s *domain.Store) {
	ctx := context.Background()
	id0, err := s.Add(
		ctx, "Release the D* demo", "ship it. ship it. ship it.",
		time.Now().Add(time.Hour), time.Time{},
	)
	if err != nil {
		panic(err)
	}
	err = s.Edit(ctx, id0, func(t *domain.Todo) error {
		t.Status = domain.StatusDone
		return nil
	})
	if err != nil {
		panic(err)
	}

	_, err = s.Add(
		ctx, "Do the laundries", "",
		time.Now().Add(-time.Hour), time.Now().Add(-time.Hour),
	)
	if err != nil {
		panic(err)
	}

	_, err = s.Add(
		ctx, "Go shopping", "- onions\n- sausages\n- bananas\n- a new broom",
		time.Now().Add(-24*time.Minute), time.Now().Add(2*time.Second),
	)
	if err != nil {
		panic(err)
	}

	_, err = s.Add(
		ctx, "Check emails", "",
		time.Now().Add(-10*time.Second), time.Now().Add(4*24*time.Hour),
	)
	if err != nil {
		panic(err)
	}

	_, err = s.Add(
		ctx, "Add more Lorem Ipsum text", `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Ut eu interdum leo, et lobortis purus. Nullam eget mi placerat, ultricies mauris quis, pulvinar magna. Phasellus ornare nulla tempor nulla tincidunt feugiat. Praesent nec molestie leo, porta tempor nibh. Quisque quis pellentesque ligula. Nunc nec diam a nisi tempor facilisis in sit amet ex. Sed in enim ut est egestas ultrices et ut massa. Praesent mattis quam ut pretium commodo. Nullam eget scelerisque est, semper viverra enim. Nam id egestas sem. Duis et pharetra tortor. Sed cursus bibendum eros, ac suscipit ante rhoncus at. Praesent ut ligula a est pharetra dapibus a a libero. Etiam fermentum quam sit amet augue pharetra scelerisque. Integer finibus sed urna quis finibus.

Mauris sapien mauris, hendrerit et purus nec, mollis blandit tellus. In metus tortor, auctor eu imperdiet ac, pharetra quis quam. In at malesuada nibh. Donec iaculis id elit dignissim maximus. Fusce dolor tortor, accumsan eu urna quis, bibendum egestas tellus. Suspendisse vulputate fringilla condimentum. Proin interdum finibus laoreet. Morbi odio sem, fringilla venenatis lorem facilisis, pulvinar molestie augue.`,
		time.Now().Add(-12*24*time.Hour), time.Now().Add(60*24*time.Hour),
	)
	if err != nil {
		panic(err)
	}
}
