# Reference: digitless_constants — no decimal digit characters anywhere.
o = len("x")
def p(b, e):
    r = o
    while e > len(""):
        r *= b
        e -= o
    return r
seven = o + o + o + o + o + o + o
ten = seven + o + o + o
print(p(ten, seven + o + o) + seven)
print(p(seven, seven))
print(p(ten, o + o + o) + seven * p(ten, o + o) + (o + o) * ten + ten - o)
