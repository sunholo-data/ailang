# Reference: bytecode_vm_trace — gcd(252,105) by subtraction
# Semantics: SUB pops b (top) then a, pushes a-b. JMPGT pops b then a, jumps if a > b.
# Jump offsets are relative to the index of the jump instruction itself.
# Every executed instruction counts toward steps, including HALT and taken/untaken jumps.
prog = [
    ("PUSH", 252),   # 0
    ("PUSH", 105),   # 1
    ("OVER", None),  # 2  loop start
    ("OVER", None),  # 3
    ("SUB", None),   # 4
    ("JMPZ", 12),    # 5  -> 17 when a-b == 0
    ("OVER", None),  # 6
    ("OVER", None),  # 7
    ("JMPGT", 4),    # 8  -> 12 when a > b
    ("OVER", None),  # 9   b = b - a branch
    ("SUB", None),   # 10
    ("JMP", -9),     # 11 -> 2
    ("SWAP", None),  # 12  a = a - b branch
    ("OVER", None),  # 13
    ("SUB", None),   # 14
    ("SWAP", None),  # 15
    ("JMP", -14),    # 16 -> 2
    ("POP", None),   # 17
    ("HALT", None),  # 18
]
stack = []
pc = 0
steps = 0
while True:
    op, arg = prog[pc]
    steps += 1
    if op == "PUSH": stack.append(arg); pc += 1
    elif op == "ADD": b=stack.pop(); a=stack.pop(); stack.append(a+b); pc += 1
    elif op == "SUB": b=stack.pop(); a=stack.pop(); stack.append(a-b); pc += 1
    elif op == "MUL": b=stack.pop(); a=stack.pop(); stack.append(a*b); pc += 1
    elif op == "DUP": stack.append(stack[-1]); pc += 1
    elif op == "OVER": stack.append(stack[-2]); pc += 1
    elif op == "SWAP": stack[-1], stack[-2] = stack[-2], stack[-1]; pc += 1
    elif op == "POP": stack.pop(); pc += 1
    elif op == "JMP": pc = pc + arg
    elif op == "JMPZ":
        v = stack.pop()
        pc = pc + arg if v == 0 else pc + 1
    elif op == "JMPGT":
        b = stack.pop(); a = stack.pop()
        pc = pc + arg if a > b else pc + 1
    elif op == "HALT": break
for v in stack: print(v)
print(f"steps: {steps}")
