# Development hints

Bag of things useful when working on the project.

## Updating dependencies

1. Manually go over repositories in `WORKSPACE` and update to the latest
   versions.

1. Update go deps with

   ```sh
   bazel run @go_sdk//:bin/go -- get -u all
   ```

   followed by running mod tidy and updating the `repositories.bzl` using

   ```sh
   bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=repositories.bzl%go_repositories -prune
   ```

   The most up-to-date commands to do that are in
   [ci.yaml](../.github/workflows/ci.yaml) which verifies that all of those are
   in sync.

   From time to time you might also want to manually go through the deps and
   check whatever there are some new major versions released: this will require
   changes to the code though.

1. Verify that all builds and passes tests. Pay attention to any deprecation
   warnings that might show up in the output because the interface of some rules
   changed.

## Debugging Cloud IoT integration

This assumes environment setup as in [../infra/README.md](../infra/README.md).

The steps below are useful for debugging some parts of Cloud IoT integration.
For example

- See raw messages on pubsub topic as send by deamon
- Publish data to Cloud IoT and see how it shows up in pubsub

1. Create keys and add a device:

   ```sh
   openssl genpkey -algorithm RSA -out rsa_private.pem -pkeyopt rsa_keygen_bits:2048
   openssl rsa -in rsa_private.pem -pubout -out rsa_public.pem
   gcloud --project=$PROJECT_ID iot devices create --region=$REGION \
      --registry=sensors --public-key path=rsa_public.pem,type=rsa-pem testing-device
   ```

1. To create, pull and remove pubsub subscription:

   ```sh
   gcloud --project=$PROJECT_ID pubsub subscriptions create --topic=measurements test-measurements-subscription
   gcloud --project=$PROJECT_ID pubsub subscriptions pull --auto-ack --limit=10 test-measurements-subscription
   gcloud --project=$PROJECT_ID pubsub subscriptions delete test-measurements-subscription
   ```

1. To send a message we need to create JWTs, in Debian `python3-jwt` package
   provides `pyjwt3` binary:

   ```sh
   curl -X POST \
      -H "authorization: Bearer $(pyjwt3 \
         --key="$(cat rsa_private.pem)" \
         --alg=RS256 \
         encode aud=${PROJECT_ID} exp=+10 iat=$(date +%s) \
      )" \
      -H 'cache-control: no-cache' \
      -H 'content-type: application/json' \
      --data "{\"binary_data\": \"$(base64 <<< '{"message": "test"}')\"}" \
      "https://cloudiotdevice.googleapis.com/v1/projects/${PROJECT_ID}/locations/${REGION}/registries/sensors/devices/testing-device:publishEvent"
   ```
