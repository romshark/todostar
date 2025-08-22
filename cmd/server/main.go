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
	add := func(
		title, description string, now, due time.Time, edit func(*domain.Todo) error,
	) (id int64) {
		ctx := context.Background()
		id, err := s.Add(
			ctx, title, description,
			now, due,
		)
		if err != nil {
			panic(err)
		}
		if edit != nil {
			if err = s.Edit(ctx, id, edit); err != nil {
				panic(err)
			}
		}
		return id
	}

	now := time.Now()

	add("Release the D* demo", "ship it. ship it. ship it.",
		now.Add(time.Hour), time.Time{},
		func(t *domain.Todo) error {
			t.Status = domain.StatusDone
			return nil
		},
	)

	add("Do the laundries", "",
		now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)

	add("Go shopping", "- onions\n- sausages\n- bananas\n- a new broom",
		now.Add(-24*time.Minute), now.Add(2*time.Second), nil,
	)

	add("Check emails", "",
		now.Add(-10*time.Second), now.Add(4*24*time.Hour), nil,
	)

	add("Add more Lorem Ipsum text",
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Ut eu interdum leo, et lobortis purus. Nullam eget mi placerat, ultricies mauris quis, pulvinar magna. Phasellus ornare nulla tempor nulla tincidunt feugiat. Praesent nec molestie leo, porta tempor nibh. Quisque quis pellentesque ligula. Nunc nec diam a nisi tempor facilisis in sit amet ex. Sed in enim ut est egestas ultrices et ut massa. Praesent mattis quam ut pretium commodo. Nullam eget scelerisque est, semper viverra enim. Nam id egestas sem. Duis et pharetra tortor. Sed cursus bibendum eros, ac suscipit ante rhoncus at. Praesent ut ligula a est pharetra dapibus a a libero. Etiam fermentum quam sit amet augue pharetra scelerisque. Integer finibus sed urna quis finibus.

Mauris sapien mauris, hendrerit et purus nec, mollis blandit tellus. In metus tortor, auctor eu imperdiet ac, pharetra quis quam. In at malesuada nibh. Donec iaculis id elit dignissim maximus. Fusce dolor tortor, accumsan eu urna quis, bibendum egestas tellus. Suspendisse vulputate fringilla condimentum. Proin interdum finibus laoreet. Morbi odio sem, fringilla venenatis lorem facilisis, pulvinar molestie augue.`,
		now.Add(-12*24*time.Hour), now.Add(60*24*time.Hour), nil,
	)

	// Archive
	add("A very old task long done", "",
		now.Add(-(100 * 24 * time.Hour)), now.Add(-(98 * 24 * time.Hour)),
		func(t *domain.Todo) error {
			t.Status, t.Archived = domain.StatusDone, true
			return nil
		})
	add("Old task that has never been done", "",
		now.Add(-(30 * 24 * time.Hour)), now.Add(-(10 * 24 * time.Hour)),
		func(t *domain.Todo) error {
			t.Archived = true
			return nil
		})
}
