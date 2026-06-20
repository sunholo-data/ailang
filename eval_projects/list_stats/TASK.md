# Task: implement the average function

`nums.ail` defines `sumList` but is missing `avg`. Implement `avg(xs: [int]) -> int` in
`nums.ail` — the integer mean of the list (sum divided by the number of elements) — and
export it so `main.ail` can use it.

AILANG has no `for`/`while` loops: count the elements with recursion (see `sumList`).

Running `main` must print `30` (the mean of [10, 20, 30, 40, 50] = 150 / 5).
