# Task: find and fix the off-by-one validator bug

This project has 25 rule modules (`rule00.ail` … `rule24.ail`). Each `ruleNN(x)` is supposed to
accept any value **at or above** its threshold NN (i.e. `x >= NN`). `main.ail` checks that every
rule accepts its own threshold value and prints how many do — it **should print 25**.

Exactly one rule module has an off-by-one bug: it uses `>` instead of `>=`, so it rejects a value
equal to its threshold. Find the buggy module and fix it so that running `main` prints `25`.
