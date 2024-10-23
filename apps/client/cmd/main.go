package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/egsam98/wow/internal/api"
	"github.com/egsam98/wow/internal/envconf"
)

const envPath = ".env"

type Envs struct {
	Addr   string `envconfig:"ADDR" required:"true"`
	Logger struct {
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
		log.Fatal().Stack().Err(err).Msgf(`Dial "Words of Wisdom" server on %s`, envs.Addr)
	}
}

func run(ctx context.Context, envs Envs) error {
	log.Info().Str("addr", envs.Addr).Msgf("Connecting to Words of Wisdom")
	client, err := api.Dial(envs.Addr)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Error().Stack().Err(err).Msg("Close client")
		}
	}()

	log.Info().Msgf("Obtaining random phrase...")
	res, err := client.Phrase(ctx)
	if err != nil {
		return err
	}
	log.Info().Str("author", res.Author).Msg(res.Quote)
	return nil
}
