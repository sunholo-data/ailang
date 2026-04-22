function branch(left, value, right) { return { type: "branch", left, value, right }; }
function leaf(value) { return { type: "leaf", value }; }

function treeSum(tree) {
  if (tree.type === "leaf") return tree.value;
  return tree.value + treeSum(tree.left) + treeSum(tree.right);
}

const tree = branch(
  branch(leaf(1), 3, leaf(4)),
  5,
  branch(null, 8, leaf(10))
);

function treeSumFull(t) {
  if (t === null) return 0;
  if (t.type === "leaf") return t.value;
  return t.value + treeSumFull(t.left) + treeSumFull(t.right);
}

console.log(treeSumFull(tree));
