To be implemented:

1. justfile
  - lint: go fmt and vet
  - vendor: go tidy and vendor
  - dev: go run
  - install: go install
2. cli/core.go
  - Data types and core logic for performing the work.
  - File redaing and writing
3. cli/main.go
  - Definitions of commands and subcommands
  - Argument parsing

Add dependencies using `go get`, then run justfile's vendor recipe.
