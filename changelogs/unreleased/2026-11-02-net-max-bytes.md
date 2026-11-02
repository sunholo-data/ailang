### Added — `--net-max-bytes` and `AILANG_NET_MAX_BYTES` for `run` and `serve-api` (2026-11-02)

The Net response-body cap was a fixed 5 MB (`NewNetContext`) with no flag or variable, so every
`httpGet`/`httpPost`/`httpRequest*` (and std/web) response over 5 MiB failed with
`BodyTooLarge("5242880")`. A hosted serve-api fetching user files by signed URL could not accept one
larger than that, whatever its upload limit or plan said (docparse advertises 10/25/50 MB).

`--net-max-bytes <size>` on `ailang run` and `ailang serve-api` sets it; `AILANG_NET_MAX_BYTES` sets it
when the flag is absent; unset keeps 5 MB. Sizes use the same parser as `--fs-max-bytes` (a byte count
or K/M/G/T suffix, `50MB`). Zero, negative or malformed is an error, and the flag without `--caps Net`
is refused rather than ignored. Over the cap, the error carries the configured number. `run --policy`
refuses the flag by name; restricted policy runs do not inherit the variable.

The body is still read whole into memory (the bytes and the `body` string are both held), so budget at
least twice the cap per in-flight response.
