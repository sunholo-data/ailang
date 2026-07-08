# Reference: glob_match_spec
# * = any run (incl empty) of non-/ chars; ? = one non-/ char;
# [...] = one char from set (ranges ok, leading ! negates; class CAN match / only if listed);
# full-string anchored match. Patterns are guaranteed well-formed. No escapes.
import sys
CASES = [
    ("*.txt", "notes.txt", None),
    ("*.txt", "dir/notes.txt", None),   # trap: fnmatch says True
    ("a*b*c", "axbxbc", None),
    ("a*b*c", "axbxbd", None),
    ("?at", "cat", None),
    ("?at", "/at", None),               # ? excludes /
    ("[a-c]at", "bat", None),
    ("[!a-c]at", "bat", None),
    ("[!a-c]at", "rat", None),
    ("[!/]x", "/x", None),              # negated class still can't match / here
    ("a[/]b", "a/b", None),             # / explicitly listed -> matches
    ("*", "", None),
    ("*x*", "xx", None),
    ("*ab", "aab", None),               # backtracking to non-greedy position
    ("a?c*", "abc", None),              # trailing * matches empty
    ("*/", "abc/", None),               # * stops at /, then literal /
]
def parse_class(pat, i):
    # pat[i] == '[' ; returns (set_of_chars, negated, next_index)
    i += 1
    neg = False
    if pat[i] == "!":
        neg = True; i += 1
    chars = set()
    while pat[i] != "]":
        if i + 2 < len(pat) and pat[i+1] == "-" and pat[i+2] != "]":
            for c in range(ord(pat[i]), ord(pat[i+2]) + 1):
                chars.add(chr(c))
            i += 3
        else:
            chars.add(pat[i]); i += 1
    return chars, neg, i + 1

def match(pat, s, pi=0, si=0):
    if pi == len(pat):
        return si == len(s)
    c = pat[pi]
    if c == "*":
        # try consuming 0..k non-/ chars
        k = si
        while True:
            if match(pat, s, pi + 1, k):
                return True
            if k < len(s) and s[k] != "/":
                k += 1
            else:
                return False
    if si == len(s):
        return False
    if c == "?":
        return s[si] != "/" and match(pat, s, pi + 1, si + 1)
    if c == "[":
        chars, neg, npi = parse_class(pat, pi)
        ok = (s[si] not in chars) if neg else (s[si] in chars)
        # negated class must still not match '/' unless... spec: [!...] matches any char
        # NOT in set. '/' not in set -> matches '/'. Keep simple + explicit: negated class
        # DOES match '/' if '/' not listed? That contradicts case 10. PIN: negated class
        # never matches '/'.
        if neg and s[si] == "/":
            ok = False
        return ok and match(pat, s, npi, si + 1)
    return s[si] == c and match(pat, s, pi + 1, si + 1)

for pat, s, _ in CASES:
    print(1 if match(pat, s) else 0)
# cross-check the fnmatch trap actually traps
import fnmatch
diffs = sum(1 for pat, s, _ in CASES if fnmatch.fnmatchcase(s, pat) != match(pat, s))
print(f"fnmatch diverges on {diffs} cases", file=sys.stderr)
