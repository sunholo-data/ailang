function isBalanced(s) {
  let count = 0;
  for (const ch of s) {
    if (ch === '(') count++;
    else if (ch === ')') count--;
    if (count < 0) return false;
  }
  return count === 0;
}

console.log(isBalanced("(())"));
console.log(isBalanced("(()"));
console.log(isBalanced("())()" ));
console.log(isBalanced("()(())"));
