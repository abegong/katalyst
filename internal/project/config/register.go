package config

// Blank-import the check-type families so config.Load can parse each configured
// check through the checks registry: under Path A the loader validates checks at
// load time, which needs their parsers registered. checks (and checks/all) no
// longer import config, so this does not cycle.
import _ "github.com/abegong/katalyst/internal/checks/all"
