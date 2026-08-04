module github.com/onklave/template-container

// Keep this in step with the builder image in the Dockerfile
// (`golang:1.26-alpine`) — that image is the source of truth for the Go version
// this service is built with. The directive is not just a language floor: it
// sets the GODEBUG compatibility baseline compiled into the binary, so trailing
// the builder bakes in opt-outs from newer hardening defaults.
go 1.26
