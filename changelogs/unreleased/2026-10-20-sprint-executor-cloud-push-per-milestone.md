### Changed — Sprint executor cloud milestone durability

- Publish every cloud task milestone through an exact task-branch gate, with bounded,
  noninteractive Git, tracking ref setup, verified pushes and safe retry resume.
- Document non-fatal publication failures and prohibit provisioning missing toolchains.
- Cover branch safety, shallow-clone wrapper decisions and retries with hermetic Git tests.
