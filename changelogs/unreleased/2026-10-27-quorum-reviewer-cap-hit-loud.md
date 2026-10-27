### Fixed — Quorum reviewer output-cap diagnostics

Failed reviewer responses now report inferred truncation when the provider omits
its finish reason and the output count reaches the positive declared token budget.
Diagnostics distinguish inference from provider-reported truncation, name the
models.yml entry, and distinguish consumed budgets from provider or context limits.
Valid votes at the cap remain present; invalid responses keep their absence reason
and salvaged reject vetoes.
