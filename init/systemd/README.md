# systemd init

1. Put the `gbcsdpd` binary in `/usr/local/bin`
1. Create a `/usr/local/etc/gbcsdpd/config.toml` configuration file
1. Copy `gbcsdpd.service` to `/usr/local/lib/systemd/system`
1. Enable service `systemctl enable gbcsdpd.service`
1. Start service `systemctl start gbcsdpd.service`

## Detailed instructions with example

All commands below are executed from the root of the repository.

1. Put the `gbcsdpd` binary in `/usr/local/bin`:

   First we have to build it, when building with bazel:

   ```
   $ bazel build //cmd/gbcsdpd
   INFO: Analyzed target //cmd/gbcsdpd:gbcsdpd (0 packages loaded, 0 targets configured).
   INFO: Found 1 target...
   Target //cmd/gbcsdpd:gbcsdpd up-to-date:
     bazel-bin/cmd/gbcsdpd/gbcsdpd_/gbcsdpd
   INFO: Elapsed time: 0.502s, Critical Path: 0.03s
   INFO: 1 process: 1 internal.
   INFO: Build completed successfully, 1 total action
   $ sudo mkdir -p /usr/local/bin
   $ sudo cp bazel-bin/cmd/gbcsdpd/gbcsdpd_/gbcsdpd /usr/local/bin/
   ```

   When building with standard go toolchain:

   ```
   $ go build ./cmd/gbcsdpd
   $ sudo mkdir -p /usr/local/bin
   $ sudo cp gbcsdpd /usr/local/bin/
   ```

1. Create a `/usr/local/etc/gbcsdpd/config.toml` configuration file.

   I'm assuming there is `config.toml` file in the root of repository with some
   working config. The example config below is for mosquitto MQTT broker with
   default configuration working on localhost.

   ```
   $ cat config.toml
   [[sinks.mqtt]]
   topic = "/measurements"
   format = "JSON"
   server_name = "localhost"
   server_port = 1883
   enable_tls = false
   $ sudo mkdir -p /usr/local/etc/gbcsdpd
   $ sudo cp config.toml /usr/local/etc/gbcsdpd/
   ```

1. Copy `gbcsdpd.service` to `/usr/local/lib/systemd/system`

   ```
   $ sudo mkdir -p /usr/local/lib/systemd/system
   $ sudo cp init/systemd/gbcsdpd.service /usr/local/lib/systemd/system/
   ```

1. Enable service `systemctl enable gbcsdpd.service`

   ```
   $ sudo systemctl enable gbcsdpd.service
   Created symlink /etc/systemd/system/multi-user.target.wants/gbcsdpd.service → /usr/local/lib/systemd/system/gbcsdpd.service.
   ```

1. Start service `systemctl start gbcsdpd.service`

   ```
   $ sudo systemctl start gbcsdpd.service
   $ systemctl status gbcsdpd.service
   ● gbcsdpd.service - gbcsdpd Daemon
        Loaded: loaded (/usr/local/lib/systemd/system/gbcsdpd.service; enabled; vendor preset: enabled)
        Active: active (running) since Wed 2021-11-24 21:42:44 CET; 5s ago
      Main PID: 12936 (gbcsdpd)
         Tasks: 7
        Memory: 4.9M
           CPU: 66ms
        CGroup: /system.slice/gbcsdpd.service
                └─12936 /usr/local/bin/gbcsdpd -config /usr/local/etc/gbcsdpd/config.toml -logtime false

   Nov 24 21:42:44 machine systemd[1]: Started gbcsdpd Daemon.
   ```

With the `config.toml` as created above and mosquitto on localhost, we can see
measurements send to the MQTT broker with:

```
$ mosquitto_sub -t /measurements
{"measurements":[{"sensorMac":"d1:e3:69:a5:a5:74","temperature":22.109999,"humidity":64.5825,"pressure":967.41,"batteryVoltage":2.977}]}
{"measurements":[{"sensorMac":"01:15:15:14:01:d1","temperature":23.9,"humidity":60.032497,"pressure":967.06,"batteryVoltage":2.281}]}
{"measurements":[{"sensorMac":"f0:af:6c:e1:a1:d5","temperature":23.779999,"humidity":58.4575,"pressure":967.7,"batteryVoltage":2.989}]}
```
