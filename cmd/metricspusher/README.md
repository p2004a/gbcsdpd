# metricspusher

This package provides an image for Cloud Run that receives Cloud Pub/Sub
messages containing serialized
[`gbcsdpd.api.v1.MeasurementsPublication`](../../api/climate.proto) and pushes
them to
[Cloud Monitoring Custom Metrics](https://cloud.google.com/monitoring/custom-metrics):
`custom.googleapis.com/sensor/measurement/{temperature,humidity,pressure,battery}`.

See sources in [infra/](../../infra) for details about usage.
