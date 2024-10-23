package envconf

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

func Load(out any, path string) error {
	if _, err := os.Stat(path); err == nil {
		if err := godotenv.Load(path); err != nil {
			return err
		}
	}
	err := envconfig.Process("", out)
	return errors.Wrap(err, "failed to load ENV-configuration")
}
