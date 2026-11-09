### Fixed — Cloud Trace deprecation notice no longer pollutes command output (#2151)

- Any `ailang` command run with `GOOGLE_CLOUD_PROJECT` set printed the Cloud Trace exporter library's
  deprecation notice as the first stderr line. The eval harness sets that variable for agent children,
  so the notice was the first line of every `ailang check` and `ailang run` result in every agent eval,
  and it appeared in fine-tune training trajectories.
- The global logger now drops that one line (`internal/platform/otel/logfilter.go`); every other log line
  is unchanged. Migrating the exporter to OTLP is still to do.
