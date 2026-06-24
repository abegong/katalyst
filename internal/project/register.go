package project

// Blank-import the check-type families so the loader (Load) can parse each
// configured check through the checks registry: it validates checks at load
// time, which needs their parsers registered. checks (and checks/all) do not
// import project, so this does not cycle.
import _ "github.com/abegong/katalyst/internal/checks/all"
