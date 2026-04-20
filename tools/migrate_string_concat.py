#!/usr/bin/env python3
"""
Migrate AILANG string `++` concatenation to `${...}` interpolation.

Approach:
- For each line, scan for `++` operators at the current nesting level.
- For each `++`, find the immediately-preceding atomic expression (left operand)
  and the immediately-following atomic expression (right operand).
- Extend the chain greedily while adjacent `++` at same nesting level.
- If the chain contains at least one string literal segment, rewrite it to a
  single `"..."` string with `${...}` interpolations.
"""
import re
import sys
from pathlib import Path


_IDENT_RE = re.compile(r'[A-Za-z_][A-Za-z0-9_]*')
_NUM_RE = re.compile(r'[0-9][0-9_.]*')
_STR_RE = re.compile(r'"(?:[^"\\]|\\.)*"')


def skip_ws(s: str, i: int) -> int:
    while i < len(s) and s[i] in ' \t':
        i += 1
    return i


def match_string_literal(s: str, i: int):
    """If s[i] starts a string literal, return index just past it; else None."""
    if i >= len(s) or s[i] != '"':
        return None
    j = i + 1
    while j < len(s):
        c = s[j]
        if c == '\\' and j + 1 < len(s):
            j += 2
            continue
        if c == '"':
            return j + 1
        j += 1
    return None  # unterminated


def match_balanced(s: str, i: int, open_c: str, close_c: str):
    """If s[i] == open_c, return index just past matching close. Else None."""
    if i >= len(s) or s[i] != open_c:
        return None
    depth = 1
    j = i + 1
    while j < len(s):
        c = s[j]
        if c == '"':
            end = match_string_literal(s, j)
            if end is None:
                return None
            j = end
            continue
        if c == open_c:
            depth += 1
        elif c == close_c:
            depth -= 1
            if depth == 0:
                return j + 1
        j += 1
    return None


def parse_atom_forward(s: str, i: int):
    """
    Starting at i, parse one atomic expression and return (start, end).
    An atom is: string literal | number | identifier (+ optional call/index/field chain)
             | parenthesized (...) | bracketed [...] | record {...}
    Returns None if no atom.
    """
    i = skip_ws(s, i)
    if i >= len(s):
        return None
    c = s[i]
    start = i
    end = None

    # String
    e = match_string_literal(s, i)
    if e is not None:
        end = e
    # Parenthesized / bracketed / brace
    elif c == '(':
        e = match_balanced(s, i, '(', ')')
        if e is None:
            return None
        end = e
    elif c == '[':
        e = match_balanced(s, i, '[', ']')
        if e is None:
            return None
        end = e
    elif c == '{':
        e = match_balanced(s, i, '{', '}')
        if e is None:
            return None
        end = e
    # Number
    elif c.isdigit() or (c == '-' and i + 1 < len(s) and s[i + 1].isdigit()):
        j = i + 1 if c == '-' else i
        m = _NUM_RE.match(s, j)
        if m is None:
            return None
        end = m.end()
    # Identifier
    elif c.isalpha() or c == '_':
        m = _IDENT_RE.match(s, i)
        if m is None:
            return None
        end = m.end()
    else:
        return None

    # Trailing call/index/field chain: "(", "[", ".ident"
    while end is not None and end < len(s):
        ch = s[end]
        if ch == '(':
            e2 = match_balanced(s, end, '(', ')')
            if e2 is None:
                break
            end = e2
        elif ch == '[':
            e2 = match_balanced(s, end, '[', ']')
            if e2 is None:
                break
            end = e2
        elif ch == '.' and end + 1 < len(s) and (s[end + 1].isalpha() or s[end + 1] == '_'):
            m = _IDENT_RE.match(s, end + 1)
            if m is None:
                break
            end = m.end()
        else:
            break

    return (start, end)


def parse_atom_backward(s: str, i: int):
    """
    Given s and position i pointing JUST PAST the end of the atom we want,
    find the (start, end) of the atomic expression ending at `i`.
    Supports: string literal | number | identifier with call/index/field chain
             | (...) | [...] | {...}.
    Returns None if no atom.
    """
    # Skip trailing whitespace
    j = i
    while j > 0 and s[j - 1] in ' \t':
        j -= 1
    end = j
    if j == 0:
        return None
    c = s[j - 1]

    # If ends with ')', ']', '}' — match backward
    if c in (')', ']', '}'):
        pairs = {')': '(', ']': '[', '}': '{'}
        open_c = pairs[c]
        depth = 1
        k = j - 2
        while k >= 0:
            ch = s[k]
            # Skip string literal if we're walking past one (handle by scanning ahead)
            if ch == '"':
                # find matching open quote
                kk = k - 1
                while kk >= 0:
                    if s[kk] == '"' and (kk == 0 or s[kk - 1] != '\\'):
                        break
                    kk -= 1
                if kk < 0:
                    return None
                k = kk - 1
                continue
            if ch == c:
                depth += 1
            elif ch == open_c:
                depth -= 1
                if depth == 0:
                    # The atom starts at k — but may be preceded by ident/chain prefix.
                    start = k
                    # Back up over any preceding call chain / identifier / `.name` / ...
                    start = _extend_prefix_backward(s, start)
                    return (start, end)
            k -= 1
        return None

    # If ends with string literal
    if c == '"':
        k = j - 2
        while k >= 0:
            if s[k] == '"' and (k == 0 or s[k - 1] != '\\'):
                return (k, end)
            k -= 1
        return None

    # Identifier or number
    if c.isalnum() or c == '_':
        k = j - 1
        while k > 0 and (s[k - 1].isalnum() or s[k - 1] == '_' or s[k - 1] == '.'):
            k -= 1
        start = k
        # prefix chain like foo.bar handled by the `.` above
        return (start, end)

    return None


def _extend_prefix_backward(s: str, i: int):
    """
    Given i is start of a `(...)` `[...]` or `{...}` group, extend leftward
    over identifier(.identifier)* that forms the function/object being called.
    Returns new start.
    """
    # We want `foo.bar(...)` or `foo[i]` to be grouped as one atom.
    k = i
    # Chain: while preceded by `.ident` or `ident` (call chain)
    # But we already are at an open bracket. Go back.
    while k > 0:
        prev = s[k - 1]
        if prev.isalnum() or prev == '_':
            # scan back over identifier
            m = k - 1
            while m > 0 and (s[m - 1].isalnum() or s[m - 1] == '_'):
                m -= 1
            k = m
            continue
        if prev == '.':
            k -= 1
            continue
        if prev == ')' or prev == ']':
            # chained call; consume group
            pairs = {')': '(', ']': '['}
            open_c = pairs[prev]
            close_c = prev
            depth = 1
            m = k - 2
            found = False
            while m >= 0:
                cc = s[m]
                if cc == '"':
                    mm = m - 1
                    while mm >= 0:
                        if s[mm] == '"' and (mm == 0 or s[mm - 1] != '\\'):
                            break
                        mm -= 1
                    if mm < 0:
                        return k
                    m = mm - 1
                    continue
                if cc == close_c:
                    depth += 1
                elif cc == open_c:
                    depth -= 1
                    if depth == 0:
                        k = m
                        found = True
                        break
                m -= 1
            if not found:
                return k
            continue
        break
    return k


def string_literal_body(s: str) -> str:
    assert s.startswith('"') and s.endswith('"'), repr(s)
    return s[1:-1]


def is_string_literal(s: str) -> bool:
    s = s.strip()
    return bool(_STR_RE.fullmatch(s))


def build_interpolation(segs):
    out = ['"']
    for seg in segs:
        seg = seg.strip()
        if is_string_literal(seg):
            out.append(string_literal_body(seg))
        else:
            out.append('${' + seg + '}')
    out.append('"')
    return ''.join(out)


def find_plusplus_chains(line: str):
    """
    Return list of (start, end, new_text) replacements for the line.
    Chains are maximal runs of `atom ++ atom [++ atom ...]` where at least one
    atom is a string literal.
    """
    # Find positions of ++ operators outside strings
    positions = []
    i = 0
    n = len(line)
    while i < n:
        c = line[i]
        if c == '"':
            e = match_string_literal(line, i)
            if e is None:
                break
            i = e
            continue
        if c == '-' and i + 1 < n and line[i + 1] == '-':
            break  # start of line comment
        if c == '+' and i + 1 < n and line[i + 1] == '+':
            positions.append(i)
            i += 2
            continue
        i += 1

    if not positions:
        return []

    # Group positions into maximal chains: adjacent `++` ops whose atoms
    # are immediately adjacent with only whitespace + ++ between.
    # We'll process one at a time, extending greedily.
    replacements = []
    consumed_ranges = []

    def overlaps(lo, hi):
        for (a, b) in consumed_ranges:
            if not (hi <= a or lo >= b):
                return True
        return False

    for pos in positions:
        if overlaps(pos, pos + 2):
            continue
        # Parse left atom (ending at pos) and right atom (starting at pos+2)
        left = parse_atom_backward(line, pos)
        right = parse_atom_forward(line, pos + 2)
        if left is None or right is None:
            continue
        left_start, left_end = left
        right_start, right_end = right
        segs = [line[left_start:left_end], line[right_start:right_end]]
        chain_end = right_end
        # Extend rightward
        while True:
            k = chain_end
            # skip whitespace
            while k < n and line[k] in ' \t':
                k += 1
            if k + 1 < n and line[k] == '+' and line[k + 1] == '+':
                nxt = parse_atom_forward(line, k + 2)
                if nxt is None:
                    break
                segs.append(line[nxt[0]:nxt[1]])
                chain_end = nxt[1]
            else:
                break
        # Only rewrite if any segment is a string literal
        if not any(is_string_literal(s) for s in segs):
            continue
        new_text = build_interpolation(segs)
        replacements.append((left_start, chain_end, new_text))
        consumed_ranges.append((left_start, chain_end))
    return replacements


def migrate_line(line: str) -> tuple[str, bool]:
    replacements = find_plusplus_chains(line)
    if not replacements:
        return line, False
    replacements.sort()
    out = []
    cursor = 0
    for (a, b, new) in replacements:
        out.append(line[cursor:a])
        out.append(new)
        cursor = b
    out.append(line[cursor:])
    return ''.join(out), True


def migrate_file(path: Path) -> int:
    src = path.read_text()
    out_lines = []
    n_changed = 0
    for line in src.splitlines(keepends=True):
        newline, changed = migrate_line(line)
        out_lines.append(newline)
        if changed:
            n_changed += 1
    if n_changed:
        path.write_text(''.join(out_lines))
    return n_changed


def main():
    total = 0
    touched = 0
    for p in sys.argv[1:]:
        path = Path(p)
        if not path.is_file():
            continue
        n = migrate_file(path)
        if n:
            print(f'{p}: {n} line(s) migrated')
            touched += 1
            total += n
    print(f'\nDone: {touched} file(s), {total} line(s) migrated.')


if __name__ == '__main__':
    main()
