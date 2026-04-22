function pipe(f, g) {
  return (x) => g(f(x));
}

function subtract(x, y) { return x - y; }
function negate(x) { return -x; }

const sub4 = (x) => subtract(x, 4);
const double = (x) => x * 2;

const sub4ThenDouble = pipe(sub4, double);
console.log(`Result: ${sub4ThenDouble(11)}`);
