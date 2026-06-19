module github.com/katabase-ai/katalyst/docs

go 1.25

// Hugo theme module. Tracked here (not in the application's go.mod) so
// `go mod tidy` on the app module never strips it. Managed via Hugo
// Modules (`hugo mod get`), not `go mod tidy`.
require github.com/alex-shpak/hugo-book v0.14.0 // indirect
