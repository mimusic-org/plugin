# go-plugin-http Usage Example

This directory contains a complete usage example for go-plugin-http, based on the [go-plugin](https://github.com/knqyf263/go-plugin) framework, demonstrating how to use HTTP request functionality on both the host and plugin sides.

[中文文档](./README.md)

## Directory Structure

```
example/
├── build/              # Build output directory (auto-generated)
│   ├── host            # Host executable
│   └── plugin.wasm     # WASM plugin
├── host/               # Host side example (main program)
│   └── main.go
├── plugin/             # Plugin side example (WASM plugin)
│   └── main.go
├── proto/              # Protocol definitions
│   ├── greeter.proto   # Plugin interface definition
│   └── *.pb.go         # Generated code (auto-generated)
├── gen.sh              # Proto code generation script
├── go.mod              # Go module configuration
├── Makefile            # Build script
└── README.md
```

## Quick Start

```bash
# Build and run in one command
make run

# Or step by step
make gen      # Generate proto code
make plugin   # Compile WASM plugin (using -buildmode=c-shared)
make host     # Compile Host program
make run      # Run the example
```

## Output

```
=== go-plugin-http Example ===
Plugin loaded successfully, calling Greet method...

[Plugin Log] Received request: name=World
[Plugin Log] Making HTTP request to httpbin.org...
INFO HttpLibrary.DoRequest method=GET url="https://httpbin.org/get?greeting=hello"
INFO HTTP request successful statusCode=200 bodyLen=313
[Plugin Log] HTTP request successful, status code: 200, response length: 313 bytes

Plugin returned: Hello, World! HTTP request successful, status code: 200, response length: 313 bytes

=== Example Complete ===
```

## Important Notes

### WASM Compilation Mode

Go 1.24+ requires `-buildmode=c-shared` to build a WASI reactor:

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm ./plugin/main.go
```

This generates an `_initialize` function instead of `_start`, allowing the module to remain active after initialization and support multiple calls to exported functions.

## Example Details

### Host Side (host/main.go)

Demonstrates how to:
1. Create a go-plugin plugin loader
2. Inject HTTP Library into the wazero runtime
3. Load a WASM plugin and call its methods

Key code:
```go
p, err := proto.NewGreeterPlugin(ctx, proto.WazeroRuntime(func(ctx context.Context) (wazero.Runtime, error) {
    r, err := proto.DefaultWazeroRuntime()(ctx)
    if err != nil {
        return nil, err
    }
    // Inject HTTP Library
    return r, httpexport.Instantiate(ctx, r, httpimpl.HttpLibraryImpl{})
}))
```

### Plugin Side (plugin/main.go)

Demonstrates how to:
1. Implement the go-plugin plugin interface
2. Use HTTP request functionality in the plugin
3. Call host functions for logging

Key code:
```go
import http "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin"

resp, err := http.Get("https://httpbin.org/get")
if err != nil {
    return nil, err
}
defer resp.Body.Close()
```

## Cleanup

```bash
make clean
```
