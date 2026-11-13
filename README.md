# heoscli

A command line for Denon/Marantz **HEOS** speakers on the local network:
discover them, check what is playing, control playback and volume, browse the
queue, search Tidal and SoundCloud, and sign in to a HEOS account.

## Install

```sh
go install github.com/ezeql/heoscli/cmd/heos@latest
```

Requires Go 1.27 or later.

## Usage

```sh
heos discover                          # find speakers on the network
heos status --name "Living Room"       # what is playing
heos volume set 30 --ip 192.168.1.10   # by address instead of name
heos queue --max 20
heos search tidal "artist" --play
heos config set defaultDevice "Living Room"   # then --name can be left out
```

`--format plain|json|tsv` picks the output. `heos --help` lists every command.

Discovery uses SSDP and falls back to scanning the local /24 for the HEOS
control port (1255). On macOS the firewall must let the binary accept
incoming connections, or the SSDP replies never arrive.

## Development

```sh
make run ARGS="status"   # run the CLI
make lint                # golangci-lint v2 (incl. gofumpt)
make race                # tests with the race detector
```

The tests are end to end: they run the real commands against a fake HEOS
device on a loopback port.

## License

[MIT](LICENSE)
