"""The Lorehallow emblem, 32×32: a lit lantern hanging from the curled tip
of a leafy branch, over a dusk-blue disc. Source of the favicons and the
app icon. Light from the top-left; the disc, branch and lantern each have
an ink outline."""

DUSK = ["deep dusk", "dusk blue", "river", "sky"]
LEAF = [".LL..", "LLll.", ".llmm", "..mm."]
LEAF_COLORS = {"L": "sunlit leaf", "l": "leaf", "m": "moss"}


def leaf(c, x, y, flip=False):
    for j, row in enumerate(LEAF):
        for i, ch in enumerate(row):
            if ch != ".":
                c.px(x + (4 - i if flip else i), y + j, LEAF_COLORS[ch])


def draw(ctx):
    sh = ctx.module("../props/shading.py")
    disc = ctx.canvas()
    sh.clump(disc, 15.5, 15.5, 15, 15, DUSK, [0.22, 0.5, 0.8])
    disc.outline("ink")

    b = ctx.canvas()
    path = [(2, 25), (5, 19), (9, 14), (14, 10), (19, 8), (24, 7), (26, 8), (27, 10), (26, 12)]
    b.polyline([(x, y + 1) for x, y in path[:6]], "bark")
    b.polyline(path[:6], "earth")
    b.polyline([(x, y - 1) for x, y in path[:6]], "dry path")
    b.polyline(path[5:], "earth")
    b.polyline([(x + 1, y) for x, y in path[5:]], "bark")
    leaf(b, 4, 13)
    leaf(b, 12, 5, flip=True)
    leaf(b, 9, 16, flip=True)
    b.outline("ink")
    disc.stamp(b)

    disc.px(22, 9, "slate")
    disc.px(22, 10, "dusk violet-grey")
    disc.stamp(ctx.pxl("../props/lantern.pxl.yaml")[0], 18, 11)
    return disc
