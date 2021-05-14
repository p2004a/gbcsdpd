# systemd init

1. Put the `gbcsdpd` binary in `/usr/local/bin`
1. Create a `/usr/local/etc/gbcsdpd/config.toml` configuration file
1. Copy `gbcsdpd.service` to `/usr/local/lib/systemd/system`
1. Enable service `systemctl enable gbcsdpd.service`
1. Start service `systemctl start gbcsdpd.service`
