# gbcsdpd Daemon

gbcsdpd daemon is a single go binary with a single configuration file that
listens for sensors BLE Advertisements via BlueZ D-Bus API and publishes them to
configured sinks. MQTT sink is the main sink, but there is also useful for
debugging stdout sink.

## Building

This repository is using [Bazel](https://bazel.build/) but it also supports
building binaries using the standard `go build` command, so both:

```
bazel build //cmd/gbcsdpd
```

and

```
go build ./cmd/gbcsdpd
```

work.

## Usage

Run the binary and it should start printing measurements on standard output.

```sh
$ ./gbcsdpd
[default-sink] 00:11:22:33:44:55 = 23.96°C, 43.17%, 963.69hPa, 2.38V
[default-sink] aa:bb:cc:dd:ee:ff = 24.55°C, 41.74%, 963.15hPa, 2.51V
[default-sink] 00:11:22:33:44:55 = 23.96°C, 43.17%, 963.69hPa, 2.39V
[default-sink] aa:bb:cc:dd:ee:ff = 24.55°C, 41.76%, 963.17hPa, 2.51V
```

`gbcsdpd` needs to be configured to be more useful. Specify path co
configuration file on `-config` flag. Configuration file is using
[TOML](https://toml.io/) format.

The default configuration for the default behavior above looks like:

```toml
adapter = "hci0"  # this is the default value that you can omit

[[sinks.stdout]]  # this sink is only added when there aren't any other defined
name = "default-sink"
```

and you can start `gbcsdpd` and load it: `./gbcsdpd -config config.toml`.

### Configuration

The only top-level setting is the Bluetooth adapter name and the rest of the
configuration consists of a list of sinks to push publications to. There can be
multiple sinks of the same and different types in the same configuration. There
are currently 3 types of sinks implemented:

- Stdout: useful for debugging, prints measurements on stdout.
- MQTT: generic MQTT target allowing to specify username, password, topic,
  format, etc.
- GCP: MQTT sink which implements custom authorization scheme required by Cloud
  IoT. Internally it's a simple wrapper over generic MQTT implementation.

Data to MQTT servers is published as
[gbcsdpd.api.v1.MeasurementsPublication](../../api/climate.proto) Protobuf
messages serialized to JSON or binary format (`format` config option on MQTT
sink).

The reference and documentation for all available configuration options is in
the [pkg/config/config_format.go](../../pkg/config/config_format.go) file.
`fConfig` type is the root of configuration.

To see how does an example configuration with a lot of options set looks like
see
[pkg/config/testdata/test1/config.toml](../../pkg/config/testdata/test1/config.toml)
file.

### Running as a service

The [init/](../../init) directory in this repository contains instructions and
configuration templates for systemd and OpenWrt init systems.

## Cross-compilation

Both Bazel and standard go distribution support cross-compilation to different
architectures.

### Bazel

See documention at https://github.com/bazelbuild/rules_go#how-do-i-cross-compile

For example arm+linux:

```
bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_arm //cmd/gbcsdpd
```

### Go Distribution

See `$GOOS` and `$GOARCH` documentation at
https://golang.org/doc/install/source#environment.

For example arm+linux:

```
GOOS=linux GOARCH=arm go build ./cmd/gbcsdpd
```
