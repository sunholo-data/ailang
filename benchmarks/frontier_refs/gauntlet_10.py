# Gauntlet master reference — 10 sections, fresh parameters.
# Each section is an isomorphic re-parameterization of a validated benchmark,
# preserving its trap structure while changing every constant.
import sys

OUT = []
def emit(s): OUT.append(str(s))

# ---- Section 1: stack VM, gcd(462, 198) by subtraction, steps counted ----
prog = [("PUSH",462),("PUSH",198),("OVER",None),("OVER",None),("SUB",None),
        ("JMPZ",12),("OVER",None),("OVER",None),("JMPGT",4),("OVER",None),
        ("SUB",None),("JMP",-9),("SWAP",None),("OVER",None),("SUB",None),
        ("SWAP",None),("JMP",-14),("POP",None),("HALT",None)]
stack, pc, steps = [], 0, 0
while True:
    op, arg = prog[pc]; steps += 1
    if op=="PUSH": stack.append(arg); pc+=1
    elif op=="SUB": b=stack.pop(); a=stack.pop(); stack.append(a-b); pc+=1
    elif op=="OVER": stack.append(stack[-2]); pc+=1
    elif op=="SWAP": stack[-1],stack[-2]=stack[-2],stack[-1]; pc+=1
    elif op=="POP": stack.pop(); pc+=1
    elif op=="JMP": pc+=arg
    elif op=="JMPZ":
        v=stack.pop(); pc = pc+arg if v==0 else pc+1
    elif op=="JMPGT":
        b=stack.pop(); a=stack.pop(); pc = pc+arg if a>b else pc+1
    elif op=="HALT": break
for v in stack: emit(v)
emit(f"steps: {steps}")

# ---- Section 2: LFU cache cap 3, LRU tie-break (keys +10, values +7 vs wave-1) ----
ops = [("put",11,17),("put",12,27),("put",13,37),("get",11,None),("put",14,47),
       ("put",15,57),("get",13,None),("get",14,None),("get",15,None),("get",11,None),
       ("put",14,51),("put",16,67),("get",15,None),("get",16,None),("get",16,None),
       ("put",17,77),("get",11,None),("get",14,None),("put",18,87),("get",17,None)]
cache = {}
for tick,(op,k,v) in enumerate(ops,1):
    if op=="get":
        if k in cache:
            cache[k][1]+=1; cache[k][2]=tick; emit(cache[k][0])
        else: emit(-1)
    else:
        if k in cache:
            cache[k][0]=v; cache[k][1]+=1; cache[k][2]=tick
        else:
            if len(cache)==3:
                victim=min(cache,key=lambda x:(cache[x][1],cache[x][2])); del cache[victim]
            cache[k]=[v,1,tick]
emit("cache: "+",".join(f"{k}={cache[k][0]}" for k in sorted(cache)))

# ---- Section 3: txn ledger (amounts x2, carol/dave) ----
bal={"carol":200,"dave":100}; snaps=[]; fails=0
for op in [("save",),("dep","carol",60),("save",),("wd","dave",160),("wd","carol",180),
           ("save",),("wd","carol",100),("dep","dave",50),("commit",),("restore",),
           ("wd","dave",150),("commit",),("dep","carol",10)]:
    if op[0]=="dep": bal[op[1]]+=op[2]
    elif op[0]=="wd":
        if bal[op[1]]>=op[2]: bal[op[1]]-=op[2]
        else:
            fails+=1
            if snaps: bal=dict(snaps.pop())
    elif op[0]=="save": snaps.append(dict(bal))
    elif op[0]=="commit":
        if snaps: snaps.pop()
        else: fails+=1
    elif op[0]=="restore":
        if snaps: bal=dict(snaps.pop())
        else: fails+=1
emit(f"carol={bal['carol']}"); emit(f"dave={bal['dave']}"); emit(f"failures={fails}")

# ---- Section 4: precedence parser (unary minus between */ and ^) ----
def tokenize(s):
    toks,i=[],0
    while i<len(s):
        if s[i].isdigit():
            j=i
            while j<len(s) and s[j].isdigit(): j+=1
            toks.append(("num",s[i:j])); i=j
        else: toks.append((s[i],s[i])); i+=1
    return toks
def parse(s):
    toks=tokenize(s); pos=[0]
    def peek(): return toks[pos[0]][0] if pos[0]<len(toks) else None
    def nxt():
        t=toks[pos[0]]; pos[0]+=1; return t
    def expr(mp):
        lhs=unary()
        while True:
            op=peek()
            if op not in ("+","-","*","/","^"): break
            p={"+":1,"-":1,"*":2,"/":2,"^":4}[op]
            if p<mp: break
            nxt()
            rhs=expr(p+1) if op!="^" else expr(p)
            lhs=f"({lhs}{op}{rhs})"
        return lhs
    def unary():
        if peek()=="-":
            nxt(); return f"(-{expr(4)})"
        return atom()
    def atom():
        t=nxt()
        if t[0]=="num": return t[1]
        if t[0]=="(":
            e=expr(1); nxt(); return e
        raise ValueError(t)
    return expr(1)
for s in ["-3^2","2^2^3","9-2-3","3^-2*5","--7+2","-5^-2^2","(2+3)*-4","200-20*-3^2"]:
    emit(parse(s))

# ---- Section 5: glob matcher (new vectors, same trap structure) ----
def pclass(pat,i):
    i+=1; neg=False
    if pat[i]=="!": neg=True; i+=1
    ch=set()
    while pat[i]!="]":
        if i+2<len(pat) and pat[i+1]=="-" and pat[i+2]!="]":
            for c in range(ord(pat[i]),ord(pat[i+2])+1): ch.add(chr(c))
            i+=3
        else: ch.add(pat[i]); i+=1
    return ch,neg,i+1
def match(pat,s,pi=0,si=0):
    if pi==len(pat): return si==len(s)
    c=pat[pi]
    if c=="*":
        k=si
        while True:
            if match(pat,s,pi+1,k): return True
            if k<len(s) and s[k]!="/": k+=1
            else: return False
    if si==len(s): return False
    if c=="?": return s[si]!="/" and match(pat,s,pi+1,si+1)
    if c=="[":
        ch,neg,npi=pclass(pat,pi)
        ok=(s[si] not in ch) if neg else (s[si] in ch)
        if neg and s[si]=="/": ok=False
        return ok and match(pat,s,npi,si+1)
    return s[si]==c and match(pat,s,pi+1,si+1)
CASES=[("*.log","server.log"),("*.log","var/server.log"),("x*y*z","xayaybz"),
       ("x*y*z","xayayb"),("?og","dog"),("?og","/og"),("[d-f]og","dog"),
       ("[!d-f]og","dog"),("[!d-f]og","hog"),("[!/]a","/a"),("b[/]c","b/c"),
       ("*",""),("*q*","qq"),("*cd","ccd"),("d?f*","def"),("*/","xyz/")]
for pat,s in CASES: emit(1 if match(pat,s) else 0)

# ---- Section 6: SSA fold (vars renamed, consts changed: 7->9, 3->4) ----
prog6=[("k","input",None),("w","input",None),("a","lit",9),("a","op",("a","-",9)),
       ("c","op",("k","*","a")),("k","op",("k","+","c")),("q","op",("w","*",4)),
       ("e","op",("k","*",9)),("f","op",("e","-","e")),("e","op",("e","+","f")),
       ("r","op",("e","*",1))]
versions={}; ssa=[]
def cur(v): return f"{v}.{versions[v]}"
for item in prog6:
    var,kind=item[0],item[1]
    if kind=="input": payload="input"
    elif kind=="lit": payload=item[2]
    else:
        l,op,rr=item[2]
        L=cur(l) if isinstance(l,str) else l
        R=cur(rr) if isinstance(rr,str) else rr
        payload=(L,op,R)
    versions[var]=versions.get(var,-1)+1
    ssa.append((cur(var),kind,payload))
table={}; kept=[]
def resolve(o):
    if isinstance(o,int): return o
    e=table[o]
    if e[0]=="const": return e[1]
    if e[0]=="copy": return resolve(e[1])
    return o
for name,kind,payload in ssa:
    if kind=="input":
        table[name]=("opaque",); kept.append((name,f"{name} = input")); continue
    if kind=="lit":
        table[name]=("const",payload); continue
    L,op,R=payload; L=resolve(L); R=resolve(R)
    if isinstance(L,int) and isinstance(R,int):
        v=L+R if op=="+" else L-R if op=="-" else L*R
        table[name]=("const",v); continue
    if op=="-" and L==R and not isinstance(L,int): table[name]=("const",0); continue
    if op=="*" and (L==0 or R==0): table[name]=("const",0); continue
    if op=="*" and R==1: table[name]=("copy",L); continue
    if op=="*" and L==1: table[name]=("copy",R); continue
    if op=="+" and R==0: table[name]=("copy",L); continue
    if op=="+" and L==0: table[name]=("copy",R); continue
    if op=="-" and R==0: table[name]=("copy",L); continue
    table[name]=("opaque",); kept.append((name,f"{name} = {L} {op} {R}"))
rres=resolve("r.0")
live=set() if isinstance(rres,int) else {rres}
out6=[]
for name,text in reversed(kept):
    if name in live:
        out6.append(text)
        for tok in text.split(" = ")[1].split(" "):
            if not tok.lstrip("-").isdigit() and tok not in ("+","-","*","input"): live.add(tok)
for line in reversed(out6): emit(line)
emit(f"return {rres}")

# ---- Section 7: LCG (multiplier 16807, seed 20260709, N=2000) ----
x=20260709; vals=[]
for _ in range(2000):
    x=(16807*x)%2147483647; vals.append(x)
emit(vals[-1]); emit(sum(sorted(vals)[-10:])); emit(sum(vals)%1000000007)

# ---- Section 8: count binary strings avoiding "0110", mod 1e9+7 ----
MOD=1000000007
# states: longest suffix that is a proper prefix of "0110": "", "0", "01", "011"
def step(state,ch):
    s=state+ch
    if s.endswith("0110"): return None
    for k in range(min(3,len(s)),-1,-1):
        if "0110".startswith(s[len(s)-k:]) if k>0 else True:
            return s[len(s)-k:] if k>0 else ""
STATES=["","0","01","011"]
idx={s:i for i,s in enumerate(STATES)}
M=[[0]*4 for _ in range(4)]
for i,s in enumerate(STATES):
    for ch in "01":
        t=step(s,ch)
        if t is not None: M[i][idx[t]]+=1
def matmul(a,b):
    return [[sum(a[i][k]*b[k][j] for k in range(4))%MOD for j in range(4)] for i in range(4)]
def matpow(m,e):
    r=[[1 if i==j else 0 for j in range(4)] for i in range(4)]
    while e:
        if e&1: r=matmul(r,m)
        m=matmul(m,m); e>>=1
    return r
def f8(n): return sum(matpow(M,n)[0])%MOD
def brute8(n):
    return sum(1 for i in range(2**n) if "0110" not in format(i,f"0{n}b"))
for n in range(1,13): assert f8(n)==brute8(n),(n,f8(n),brute8(n))
emit(f8(10**18)); emit(f8(987654321987654321))

# ---- Section 9: commonmark '*' emphasis (fresh vectors) ----
sys.path.insert(0, '/Users/mark/dev/sunholo/ailang/benchmarks/frontier_refs')
import importlib.util as _il
_spec=_il.spec_from_file_location("emph","/Users/mark/dev/sunholo/ailang/benchmarks/frontier_refs/commonmark_emphasis.py")
_m=_il.module_from_spec(_spec)
import io,contextlib
with contextlib.redirect_stdout(io.StringIO()): _spec.loader.exec_module(_m)
V9=["*river stone*","b * candle dim*","**dune**crest","***basalt***","**murmur*",
    "*cinder**","*fen**reed**mist*","*loam**peat*","***crag*ledge**","**flint**spark**",
    "*p*q*r*","twist**knot*loop"]
for v in V9: emit(_m.render(v))

# ---- Section 10: dependency resolver (versions +10) ----
UA={("app",11):[("lib",11,13),("util",13,13)],("lib",13):[("core",13,13)],
    ("lib",12):[("core",11,12)],("lib",11):[],("util",13):[("core",11,11)],
    ("core",13):[],("core",12):[],("core",11):[]}
UB={("app",11):[("x",11,11),("y",11,11)],("x",11):[("core",11,11)],
    ("y",11):[("core",12,12)],("core",11):[],("core",12):[]}
def versions_of(u,name): return sorted([v for (n,v) in u if n==name],reverse=True)
def resolve10(u):
    def go(assign,wl):
        if not wl: return assign
        (name,lo,hi),rest=wl[0],wl[1:]
        if name in assign:
            return go(assign,rest) if lo<=assign[name]<=hi else None
        for v in [v for v in versions_of(u,name) if lo<=v<=hi]:
            na=dict(assign); na[name]=v
            r=go(na,u[(name,v)]+rest)
            if r is not None: return r
        return None
    r=go({"app":11},list(UA[("app",11)]) if u is UA else list(UB[("app",11)]))
    return r
for U in (UA,UB):
    r=resolve10(U)
    if r is None: emit("conflict")
    else:
        emit("resolved")
        for k in sorted(r): emit(f"{k}={r[k]}")

print("\n".join(OUT))
print(f"TOTAL LINES: {len(OUT)}", file=sys.stderr)
