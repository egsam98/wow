package envconf

import (
	"os"

	"github.com/egsam98/errors"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Load optionally injects environment variables from provided `path` into OS and parses them into `out` struct
// (see github.com/kelseyhightower/envconfig). The `path` is optional argument
func Load(out any, path string) error {
	if _, err := os.Stat(path); err == nil {
		if err := godotenv.Load(path); err != nil {
			return err
		}
	}
	err := envconfig.Process("", out)
	return errors.Wrap(err, "failed to load ENV-configuration")
}
