// This file only marks web/ as a separate module so that Go tooling run
// from the repository root ("./...") skips the frontend and, above all,
// web/node_modules, where npm packages sometimes ship Go files.
module github.com/Sergio-dot/urtube/web

go 1.27
