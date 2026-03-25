# go-plugin-http

[![Go Reference](https://pkg.go.dev/badge/github.com/mimusic-org/plugin/pkg/go-plugin-http.svg)](https://pkg.go.dev/github.com/mimusic-org/plugin/pkg/go-plugin-http)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Provides HTTP request capabilities for WASM plugins based on [go-plugin](https://github.com/knqyf263/go-plugin).

Since the WASM environment does not support Go's standard `net/http` package, this library enables WASM plugins to make HTTP requests through host function proxying.

[中文文档](./README.md)

## Requirements

- Go 1.24+ (requires `-buildmode=c-shared` for WASI reactor support)
- [go-plugin](https://github.com/knqyf263/go-plugin) v0.9.0+
- [wazero](https://github.com/tetratelabs/wazero) v1.11.0+

## Installation

```bash
go get github.com/mimusic-org/plugin/pkg/go-plugin-http
```

## Usage

### Host Side (Main Program)

Inject HTTP Library when creating the wazero runtime:

```go
import (
    httpexport "github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
    httpimpl "github.com/mimusic-org/plugin/pkg/go-plugin-http/impl"
)

// Inject HTTP Library when creating the plugin loader
loader, err := plugins.NewPluginServicePlugin(ctx,
    plugins.WazeroRuntime(func(ctx context.Context) (wazero.Runtime, error) {
        r, err := plugins.DefaultWazeroRuntime()(ctx)
        if err != nil {
            return nil, err
        }
        // Inject HTTP Library
        if err := httpexport.Instantiate(ctx, r, httpimpl.HttpLibraryImpl{}); err != nil {
            return nil, err
        }
        return r, nil
    }))
```

### Plugin Side (WASM Plugin)

Use HTTP requests in your plugin code:

```go
import http "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin"

// GET request
resp, err := http.Get("https://example.com/api")
if err != nil {
    // Handle error
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)

// POST request
resp, err := http.Post("https://example.com/api", "application/json", bytes.NewReader(jsonData))

// Custom request
req, _ := http.NewRequest("PUT", "https://example.com/api", body)
req.Header.Set("Authorization", "Bearer token")
resp, err := http.Do(req)

// Use custom client (with timeout)
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get("https://example.com/api")
```

## API Reference

### Types

- `Header` - HTTP headers, similar to `net/http.Header`
- `Request` - HTTP request, similar to `net/http.Request`
- `Response` - HTTP response, similar to `net/http.Response`
- `Client` - HTTP client, similar to `net/http.Client`

### Functions

- `NewRequest(method, url string, body io.Reader) (*Request, error)` - Create a request
- `Get(url string) (*Response, error)` - Make a GET request
- `Post(url, contentType string, body io.Reader) (*Response, error)` - Make a POST request
- `Head(url string) (*Response, error)` - Make a HEAD request
- `Do(req *Request) (*Response, error)` - Execute a request

### Client Methods

- `Do(req *Request) (*Response, error)` - Execute a request
- `DoContext(ctx context.Context, req *Request) (*Response, error)` - Execute a request with context
- `Get(url string) (*Response, error)` - Make a GET request
- `Post(url, contentType string, body io.Reader) (*Response, error)` - Make a POST request
- `Head(url string) (*Response, error)` - Make a HEAD request

## Complete Example

See the [example](./example) directory for a complete usage example, including:

- How to load WASM plugins and inject HTTP Library on the host side
- How to use HTTP request functionality on the plugin side
- How to compile WASM plugins in WASI reactor mode

Run the example:

```bash
cd example
make run
```

## WASM Compilation Notes

Go 1.24+ requires `-buildmode=c-shared` to build a WASI reactor:

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm ./plugin/main.go
```

This generates an `_initialize` function instead of `_start`, allowing the module to remain active after initialization and support multiple calls to exported functions.

## Development

### Regenerate protobuf code

```bash
./gen.sh
```

### Run tests

```bash
cd example
make run
```

## License

Apache License 2.0 - See [LICENSE](./LICENSE) file for details
