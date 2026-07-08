# Reference: commonmark_emphasis — '*'-only emphasis via the delimiter-stack
# algorithm (flanking rules + rule of 3). Cross-validated against cmark.
import sys

PUNCT = set("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

def classify(s):
    # returns list of (kind, text) tokens: 'text' or 'run'
    toks, i = [], 0
    while i < len(s):
        if s[i] == '*':
            j = i
            while j < len(s) and s[j] == '*':
                j += 1
            toks.append(['run', s[i:j]])
            i = j
        else:
            j = i
            while j < len(s) and s[j] != '*':
                j += 1
            toks.append(['text', s[i:j]])
            i = j
    return toks

class Delim:
    def __init__(self, node, length):
        self.node = node          # index into nodes list
        self.count = length
        self.orig = length
        before = None  # set later
        self.can_open = False
        self.can_close = False

def parse(s):
    # nodes: mutable list of [kind, payload]; kind: 'text' | 'run' | 'open' | 'close'
    nodes = classify(s)
    delims = []
    pos = 0
    for idx, node in enumerate(nodes):
        if node[0] == 'run':
            start = pos
            end = pos + len(node[1])
            before = s[start - 1] if start > 0 else ' '
            after = s[end] if end < len(s) else ' '
            ws_b, ws_a = before == ' ', after == ' '
            p_b, p_a = before in PUNCT, after in PUNCT
            left_flank = (not ws_a) and ((not p_a) or ws_b or p_b)
            right_flank = (not ws_b) and ((not p_b) or ws_a or p_a)
            d = Delim(idx, len(node[1]))
            d.can_open = left_flank
            d.can_close = right_flank
            delims.append(d)
        pos += len(node[1])

    matches = []  # (opener_node, closer_node, use) in application order
    i = 0
    while i < len(delims):
        closer = delims[i]
        if not (closer.can_close and closer.count > 0):
            i += 1
            continue
        # find opener
        j = i - 1
        opener = None
        while j >= 0:
            o = delims[j]
            if o.count > 0 and o.can_open:
                # rule of 3
                blocked = (o.can_close or closer.can_open) and \
                          ((o.orig + closer.orig) % 3 == 0) and \
                          not (o.orig % 3 == 0 and closer.orig % 3 == 0)
                if not blocked:
                    opener = o
                    break
            j -= 1
        if opener is None:
            if not closer.can_open:
                delims.pop(i)   # stars stay literal; skip permanently
            else:
                i += 1
            continue
        use = 2 if (opener.count >= 2 and closer.count >= 2) else 1
        opener.count -= use
        closer.count -= use
        matches.append((opener.node, closer.node, use))
        # remove delimiters strictly between opener and closer
        oi = delims.index(opener)
        delims[oi + 1:i] = []
        i = delims.index(closer)
        if opener.count == 0:
            delims.remove(opener)
            i -= 1
        if closer.count == 0:
            delims.remove(closer)
        # i now points at the element after the removed closer (or closer itself)
    return nodes, matches

def render(s):
    nodes, matches = parse(s)
    # per node: remaining literal stars = orig len minus consumed; tags inserted
    consumed = {}   # node idx -> stars consumed
    opens = {}      # node idx -> list of tags (inner-first)
    closes = {}
    for (on, cn, use) in matches:
        tag = 'strong' if use == 2 else 'em'
        consumed[on] = consumed.get(on, 0) + use
        consumed[cn] = consumed.get(cn, 0) + use
        opens.setdefault(on, []).append(tag)
        closes.setdefault(cn, []).append(tag)
    out = []
    for idx, node in enumerate(nodes):
        if node[0] == 'text':
            out.append(node[1])
        else:
            leftover = len(node[1]) - consumed.get(idx, 0)
            # literal leftover stars attach OUTSIDE opens, INSIDE?? — per cmark:
            # leftover stars of an opener run precede the open tags; of a closer
            # run follow the close tags. A run can be both (rare in our vectors).
            # open tags: outermost first = REVERSE match order (a later match
            # on the same run wraps the earlier one). close tags: match order
            # (innermost closes first). leftover literal stars sit between
            # close tags and open tags for dual-use runs; before opens / after
            # closes otherwise.
            open_tags = ''.join(f'<{t}>' for t in reversed(opens.get(idx, [])))
            close_tags = ''.join(f'</{t}>' for t in closes.get(idx, []))
            out.append(close_tags + '*' * leftover + open_tags)
    return ''.join(out)

VECTORS = [
    "*wind mill*",
    "a * lantern glow*",
    "**tide**pool",
    "***quartz***",
    "**quiet*",
    "*ember**",
    "*sky**lark**dawn*",
    "*moss**fern*",
    "***husk*kernel**",
    "**pearl**shell**",
    "*a*b*c*",
    "escape**not*done",
]
for v in VECTORS:
    print(render(v))
