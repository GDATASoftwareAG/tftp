# A Cross-Platform TFTP Server with Configurable Request Handlers and `fs.FS` support

The TFTP Server is highly flexible for all platforms due to the Go `fs.FS` package.
It is currently used in production to install various servers via [PXE](https://en.wikipedia.org/wiki/Preboot_Execution_Environment).

The server supports different handlers for file delivery and comes with an already implemented one, which supports the `fs.FS` package.
Everything you need to know to implement your own handler or just provide your custom file system can be found [here](./cmd/serve.go)
The default handler takes an os-specific path and ensures that all requested files are within the given path. It matches on every input.
The handlers are evaluated in their order of registration, so you have to register your handlers from the most to least specific `Matches()`-function.

# Prerequisites
You need Go1.17+ and Taskfile support: `go install github.com/go-task/task/v3/cmd/task@latest`
After that just run `task bootstrap` to set up all dependencies and prepare this repository
To see all available tasks run `task help`

## Usage
This project uses the [cobra](https://github.com/spf13/cobra) package, so all you have to do is run `tftp help`.
The server provides different http endpoints for prometheus metrics or [profiling](https://pkg.go.dev/net/http/pprof). By default, they are available at:
- :9100/metrics
- :9100/debug/pprof/
- :9100/debug/pprof/cmdline
- :9100/debug/pprof/profile
- :9100/debug/pprof/symbol
- :9100/debug/pprof/trace
- :9100/debug/pprof/heap



## Config
To pass a config to the server place a `config.yaml` in one of the following folders:
- `.`
- `/etc/tftp`

If you do not provide any config files the server uses default values corresponding to the one in the [main.go](./cmd/main.go).
It is possible to overwrite the configs with environment variables.
Simply use the `TFTP_` prefix and navigate to the config you want to override. For example:
```console
TFTP_FSHANDLERBASEDIR="<your-os-specific-path>"
TFTP_TFTP_PORT=1234
TFTP_METRICS_ENABLE=false
```

## Extensions
The TFTP Server ([RFC1350](https://datatracker.ietf.org/doc/html/rfc1350/)) supports the following extensions ([RFC1782](https://datatracker.ietf.org/doc/html/rfc1782)):
- Transfer size option ([RFC2349](https://datatracker.ietf.org/doc/html/rfc2349))
- Blocksize option ([RFC2348](https://datatracker.ietf.org/doc/html/rfc2348))
