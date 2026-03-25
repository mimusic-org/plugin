module github.com/mimusic-org/plugin/pkg/go-plugin-http/example

go 1.26

require (
	github.com/mimusic-org/plugin/pkg/go-plugin-http v0.0.0
	github.com/knqyf263/go-plugin v0.9.0
	github.com/tetratelabs/wazero v1.11.0
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/sys v0.38.0 // indirect

replace github.com/mimusic-org/plugin/pkg/go-plugin-http => ../
