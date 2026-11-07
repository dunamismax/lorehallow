"""The Lorehallow wordmark, 192×64: chunky hand-drawn gold letters with an
extruded underside, a vine climbing the L and a lantern hanging from the
w. Light from the top-left, ink outline. Shown at 2x to 4x on the site.

Letters are built from strokes 5 px wide on a 32 px cap height (22 px
x-height), then shaded: lit edges facing up or left are glow, the body
lantern gold, edges facing down or right amber, with an ember and bark
extrusion below for depth.
"""

S = 5  # stroke width
CAP, XH = 32, 22  # cap height, x-height
TRACK = 2  # space between letters


def ring(g, x, y, w, h, t=S):
    """A ring: an ellipse w×h minus one inset by t."""
    from_outer = ellipse(w, h)
    inner = ellipse(w - 2 * t, h - 2 * t)
    for j in range(h):
        for i in range(w):
            ii, jj = i - t, j - t
            hole = 0 <= ii < w - 2 * t and 0 <= jj < h - 2 * t and inner[jj][ii]
            if from_outer[j][i] and not hole:
                g.add((x + i, y + j))


def ellipse(w, h):
    cx, cy = (w - 1) / 2, (h - 1) / 2
    return [[((i - cx) / (w / 2)) ** 2 + ((j - cy) / (h / 2)) ** 2 <= 1.0 for i in range(w)]
            for j in range(h)]  # fmt: skip


def rect(g, x, y, w, h):
    for j in range(h):
        for i in range(w):
            g.add((x + i, y + j))


def thick_line(g, x0, y0, x1, y1, t=S):
    n = max(abs(x1 - x0), abs(y1 - y0))
    for k in range(n + 1):
        x = round(x0 + (x1 - x0) * k / n)
        y = round(y0 + (y1 - y0) * k / n)
        rect(g, x, y, t, 1)


def cut(g, pts):
    for p in pts:
        g.discard(p)


TOP = CAP - XH  # first row of lowercase letters


def glyph_L():
    g = set()
    rect(g, 0, 0, S + 1, CAP)
    rect(g, 0, CAP - S, 15, S)
    rect(g, 13, CAP - S - 2, 2, 2)  # a little upturned foot
    cut(g, [(0, 0), (S, 0), (14, CAP - S - 2)])
    return g, 15


def glyph_o():
    g = set()
    ring(g, 0, TOP, 17, XH)
    return g, 17


def glyph_r():
    g = set()
    rect(g, 0, TOP, S, XH)
    arch = set()
    ring(arch, 0, TOP, 15, 14)
    g |= {(x, y) for x, y in arch if x >= S - 1 and y < TOP + 6}
    rect(g, 11, TOP + 3, 4, 3)
    cut(g, [(0, TOP)])
    return g, 15


def glyph_e():
    g = set()
    ring(g, 0, TOP, 17, XH)
    rect(g, 2, TOP + 9, 14, 4)
    cut(g, [(x, y) for x in range(11, 17) for y in range(TOP + 13, TOP + 16)])
    return g, 17


def glyph_h():
    g = set()
    rect(g, 0, 0, S, CAP)
    arch = set()
    ring(arch, 0, TOP, 16, 16)
    g |= {(x, y) for x, y in arch if x >= S - 1 and y < TOP + 8}
    rect(g, 11, TOP + 7, S, XH - 7)
    cut(g, [(0, 0)])
    return g, 16


def glyph_a():
    """Double-storey a: a top arch into the right stem, and a lower bowl."""
    g = set()
    arch = set()
    ring(arch, 0, TOP, 16, 14)
    g |= {(x, y) for x, y in arch if y < TOP + 5 and x >= 2}
    rect(g, 11, TOP + 4, S, XH - 4)
    bowl = set()
    ring(bowl, 0, TOP + 9, 15, XH - 9, 4)
    g |= {(x, y) for x, y in bowl if x <= 12}
    return g, 16


def glyph_l():
    g = set()
    rect(g, 0, 0, S, CAP - 2)
    rect(g, 1, CAP - 2, S + 1, 2)
    rect(g, S, CAP - 4, 2, 2)
    cut(g, [(0, 0)])
    return g, 7


def glyph_w():
    g = set()
    thick_line(g, 0, TOP, 4, CAP - 1, 4)
    thick_line(g, 4, CAP - 1, 9, TOP + 6, 4)
    thick_line(g, 9, TOP + 6, 14, CAP - 1, 4)
    thick_line(g, 14, CAP - 1, 19, TOP, 4)
    return g, 23


WORD = [glyph_L, glyph_o, glyph_r, glyph_e, glyph_h, glyph_a, glyph_l, glyph_l, glyph_o, glyph_w]


def letters():
    """The word's pixels, origin at the cap line of the first letter."""
    pts, x = set(), 0
    for fn in WORD:
        g, w = fn()
        pts |= {(gx + x, gy) for gx, gy in g}
        x += w + TRACK
    return pts, x - TRACK


def draw(ctx):
    c = ctx.canvas()
    word, width = letters()
    ox = (192 - width) // 2 - 4
    oy = 14
    word = {(x + ox, y + oy) for x, y in word}
    # extrusion: the letters' underside, down and to the right
    for x, y in word:
        for dx, dy, col in ((1, 3, "bark"), (1, 2, "ember"), (0, 1, "ember")):
            if (x + dx, y + dy) not in word:
                c.px(x + dx, y + dy, col)
    for x, y in word:
        up, left = (x, y - 1) in word, (x - 1, y) in word
        down, right = (x, y + 1) in word, (x + 1, y) in word
        if not up or not left:
            col = "glow"
        elif not down or not right:
            col = "amber"
        else:
            col = "lantern gold"
        c.px(x, y, col)
    # a few bright glints on the lit edges
    for x, y in ((ox + 1, oy + 2), (ox + 40, oy + 11), (ox + 95, oy + 11), (ox + 132, oy + 2)):
        if (x, y) in word:
            c.px(x, y, "parchment white")
    vine(c, ox, oy)
    hook(c, ox + width, oy)
    c.outline("ink")
    c.stamp(ctx.pxl("../props/lantern.pxl.yaml")[0], ox + width + 3, oy + TOP + 5)
    return c


def vine(c, ox, oy):
    """A vine climbing the L's stem, curling over its top, with leaves."""
    stem = [(ox - 3, oy + 31), (ox - 2, oy + 27), (ox + 2, oy + 24), (ox + 6, oy + 20),
            (ox + 7, oy + 15), (ox + 3, oy + 11), (ox - 1, oy + 7), (ox - 1, oy + 2),
            (ox + 3, oy - 2), (ox + 8, oy - 3), (ox + 11, oy - 1)]  # fmt: skip
    c.polyline(stem, "moss shadow")
    c.polyline([(x + 1, y) for x, y in stem], "moss")
    leaves = [(ox - 5, oy + 29, 1), (ox - 5, oy + 24, 1), (ox + 8, oy + 21, -1),
              (ox + 9, oy + 16, -1), (ox - 5, oy + 12, 1), (ox - 4, oy + 6, 1),
              (ox + 1, oy - 5, 1), (ox + 12, oy - 4, -1), (ox + 5, oy - 6, -1)]  # fmt: skip
    for x, y, d in leaves:
        leaf(c, x, y, d)


def leaf(c, x, y, d):
    """A 4×3 leaf pointing left (d=1) or right (d=-1), lit on top."""
    pts = [(0, 0), (1, 0), (2, 0), (-1, 1), (0, 1), (1, 1), (2, 1), (3, 1), (0, 2), (1, 2)]
    for px, py in pts:
        col = "sunlit leaf" if py == 0 else "leaf" if py == 1 else "moss"
        c.px(x + px * -d if d < 0 else x + px, y + py, col)


def hook(c, right, oy):
    """An iron bracket off the w's last stroke for the lantern to hang on."""
    y = oy + TOP
    c.rect(right, y, 7, 2, "slate")
    c.rect(right, y, 7, 1, "stone")
    c.rect(right + 6, y + 2, 1, 3, "dusk violet-grey")
