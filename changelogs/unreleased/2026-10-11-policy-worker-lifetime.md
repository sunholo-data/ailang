### Fixed — policy worker lifetime follows its runtime owner (2026-10-11)

Policy supervisors now register private stop and cleanup receipt channels with the same runtime executable. Their separately grouped workers receive an independent supervisor lifeline before admission, so losing the supervisor triggers bounded worker cleanup. Admission still uses its original fd-3 channel, and policy timeout/output limits still stop execution immediately.

Managed and async runtime workers require the nested cleanup receipt, direct-child reaping and owned I/O completion before successful cancellation. Missing or failed receipts remain visible failures; the existing process-group boundary remains for external commands and different runtime executables.
