function Num(n) { return { type: "Num", n }; }
function Add(l, r) { return { type: "Add", l, r }; }
function Mul(l, r) { return { type: "Mul", l, r }; }

function evaluate(expr) {
  if (expr.type === "Num") return expr.n;
  if (expr.type === "Add") return evaluate(expr.l) + evaluate(expr.r);
  if (expr.type === "Mul") return evaluate(expr.l) * evaluate(expr.r);
}

const expr = Mul(Add(Num(3), Num(4)), Add(Num(2), Num(5)));
console.log(evaluate(expr));
