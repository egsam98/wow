package main

import (
	"context"
	_ "embed"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/sync/errgroup"

	memrepo "github.com/egsam98/wow/apps/server/internal/repository/mem"
	"github.com/egsam98/wow/apps/server/internal/server"
	"github.com/egsam98/wow/internal/api"
	"github.com/egsam98/wow/internal/envconf"
	"github.com/egsam98/wow/internal/pow"
)

const envPath = ".env"

type Envs struct {
	Addr         string        `envconfig:"ADDR" required:"true"`
	PuzzleZeroes uint          `envconfig:"PUZZLE_ZEROES" default:"3"`
	TCPDeadline  time.Duration `envconfig:"TCP_DEADLINE" default:"20s"`
	Logger       struct {
		Pretty bool          `envconfig:"LOG_PRETTY" default:"false"`
		Lvl    zerolog.Level `envconfig:"LOG_LVL" default:"debug"`
	}
}

func main() {
	var envs Envs
	if err := envconf.Load(&envs, envPath); err != nil {
		log.Fatal().Err(err).Msg("Load environment variables")
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(envs.Logger.Lvl)
	if envs.Logger.Pretty {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) { w.TimeFormat = time.RFC3339 })).
			With().
			Timestamp().
			Logger()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, envs); err != nil {
		log.Fatal().Stack().Err(err).Msg("Start server")
	}
}

func run(ctx context.Context, envs Envs) error {
	puzzle, err := pow.NewPuzzle(envs.PuzzleZeroes)
	if err != nil {
		return err
	}
	repo, err := memrepo.NewRepository()
	if err != nil {
		return err
	}

	srv := api.NewServer(
		envs.Addr,
		envs.TCPDeadline,
		server.NewHandler(repo),
		puzzle,
	)
	defer srv.Close()

	g, ctx := errgroup.WithContext(ctx)
	log.Info().
		Str("addr", envs.Addr).
		Uint("puzzle_zeroes(complexity)", envs.PuzzleZeroes).
		Msg("Listening server")
	g.Go(func() error { return srv.Listen(ctx) })

	// TODO Healthcheck

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
