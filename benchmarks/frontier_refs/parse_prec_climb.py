# Reference: parse_prec_climb — precedence climbing with the math-convention twist:
# unary minus binds TIGHTER than * / but LOOSER than ^  (so -2^2 = -(2^2), -2*3 = (-2)*3)
# ^ is right-associative; + - * / left-associative.
# prec: + - = 1, * / = 2, unary - = 3, ^ = 4
INPUTS = [
    "-2^2",
    "2^3^2",
    "8-4-2",
    "2^-3*4",
    "--5+1",
    "-2^-2^3",
    "(1+2)*-3",
    "100-10*-2^2",
]
def tokenize(s):
    toks, i = [], 0
    while i < len(s):
        c = s[i]
        if c.isdigit():
            j = i
            while j < len(s) and s[j].isdigit(): j += 1
            toks.append(("num", s[i:j])); i = j
        else:
            toks.append((c, c)); i += 1
    return toks

def parse(s):
    toks = tokenize(s)
    pos = [0]
    def peek(): return toks[pos[0]][0] if pos[0] < len(toks) else None
    def next_tok():
        t = toks[pos[0]]; pos[0] += 1; return t
    # expr(min_prec) via precedence climbing over binary ops
    def parse_expr(min_prec):
        lhs = parse_unary()
        while True:
            op = peek()
            if op not in ("+", "-", "*", "/", "^"): break
            prec = {"+":1, "-":1, "*":2, "/":2, "^":4}[op]
            if prec < min_prec: break
            next_tok()
            # left assoc: rhs at prec+1 ; right assoc (^): rhs at prec
            rhs = parse_expr(prec + 1) if op != "^" else parse_expr(prec)
            lhs = f"({lhs}{op}{rhs})"
        return lhs
    def parse_unary():
        if peek() == "-":
            next_tok()
            # unary minus prec 3: its operand may contain ^ (prec 4) but not * / + -
            operand = parse_expr(4)  # only ^-level and atoms bind tighter
            return f"(-{operand})"
        return parse_atom()
    def parse_atom():
        t = next_tok()
        if t[0] == "num": return t[1]
        if t[0] == "(":
            e = parse_expr(1)
            next_tok()  # ')'
            return e
        raise ValueError(t)
    return parse_expr(1)

for s in INPUTS:
    print(parse(s))
