function foldl(f, acc, xs) {
  if (xs.length === 0) return acc;
  return foldl(f, f(acc, xs[0]), xs.slice(1));
}

const nums = [2, 4, 6, 8, 10];
const sum = foldl((a, b) => a + b, 0, nums);
const product = foldl((a, b) => a * b, 1, nums);
const max = foldl((a, b) => a > b ? a : b, 7, [2, 8, 3, 11, 5, 1]);

console.log(`Sum: ${sum}`);
console.log(`Product: ${product}`);
console.log(`Max: ${max}`);
