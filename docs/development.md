# Development hints

Bag of things useful when working on the project.

## Updating dependencies

1. Manually go over repositories in `WORKSPACE` and update to the latest
   versions.

1. Update go deps with

   ```sh
   bazel run @rules_go//go -- get -u ./...
   ```

   there might be some missed when used only in test, then you can:

   ```
   bazel run @rules_go//go -- get 'github.com/fhmq/hmq@v1.5.7'
   ```

   Make sure to follow that up with

   ```
   bazel run @rules_go//go -- mod tidy
   ```

   The `use_repo` in `MODULE.bazel` is automatically updated when we use the
   `@rules_go//go` rule.

   The most up-to-date commands to do that are in
   [ci.yaml](../.github/workflows/ci.yaml) which verifies that all of those are
   in sync.

   From time to time you might also want to manually go through the deps and
   check whatever there are some new major versions released: this will require
   changes to the code though.

1. Verify that all builds and passes tests. Pay attention to any deprecation
   warnings that might show up in the output because the interface of some rules
   changed.
