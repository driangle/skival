# Setting Up a Go Project

## Prerequisites

Install Go from [go.dev/dl](https://go.dev/dl) or via a package manager:

```sh
# macOS
brew install go

# Linux (apt)
sudo apt install golang-go
```

Verify the installation:

```sh
go version
```

## Create a New Project

```sh
mkdir myproject && cd myproject
go mod init github.com/yourname/myproject
```

This creates `go.mod`, which tracks your module name and dependencies.

## Project Structure

```
myproject/
  go.mod          # module definition and dependencies
  go.sum          # dependency checksums (auto-generated)
  main.go         # entry point for executables
  internal/       # private packages (not importable externally)
  cmd/            # additional binaries (one subdirectory per binary)
```

## Write Code

`main.go`:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

## Run and Build

```sh
go run .          # run without building a binary
go build -o myapp # compile to a binary named myapp
./myapp
```

## Manage Dependencies

```sh
go get github.com/some/package   # add a dependency
go mod tidy                      # remove unused dependencies
```

## Testing

```sh
go test ./...          # run all tests
go test -v ./...       # verbose output
go test -run TestName  # run a specific test
```

Write tests in files ending with `_test.go`:

```go
package main

import "testing"

func TestGreeting(t *testing.T) {
    got := greeting()
    if got != "Hello, world!" {
        t.Errorf("got %q, want %q", got, "Hello, world!")
    }
}
```

## Useful Commands

| Command | Description |
|---|---|
| `go fmt ./...` | Format all source files |
| `go vet ./...` | Report likely mistakes |
| `go doc <pkg>` | Show package documentation |
| `go env` | Print Go environment variables |
