### Added — Transcript secret detection

Legacy mission-controller fires now create a per-fire timestamp marker and run a bounded,
nonfatal transcript secret scan after recording their slot verdict. The detector reads literal
assignments without executing them, checks fresh files for literal and common JSON-escaped
values, and reports key names and paths only. A leak sends one fire-specific notice: key
names and paths go to the private controlplane inbox, and the public bookkeeping issue gets a
hit count only;
instrument failures remain explicit and preserve the controller result.

Synthetic tests cover parsing, file selection, subprocess failures, cleanup, and extracted
driver wiring. Binary `ailang mission iterate` fires bypass this legacy insertion point.
This detects exposure after writing; it does not prevent, redact, or rotate credentials.
