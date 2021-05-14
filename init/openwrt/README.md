# OpenWrt init script

1. Make sure you have `bluez-daemon` package installed on the device. You might
   want to set `AutoEnable` to true in `/etc/bluetooth/main.conf` as it's off by
   default.

1. Optional: Install `ca-bundle` and `ca-certificates` packages if you don't use
   TLS with self-signed certificates on the MQTT server and don't want to
   provide root certificates chain yourself.

1. Build for correct architecture as described in
   [cmd/gbcsdpd](../../cmd/gbcsdpd). For example:

   ```sh
   bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_arm //cmd/gbcsdpd
   ```

1. Copy the binary to `/usr/bin` on the OpenWrt router.

1. Create `/etc/gbcsdpd/config.toml` configuration.

1. Copy the `gbcsdpd` init service file in this directory to `/etc/init.d/`.

1. Enable service `service gbcsdpd enable`.

1. Start service `service gbcsdpd start`.
