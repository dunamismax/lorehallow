"""Shared code for the 16-tile transition sets (STYLE.md "Transition sets").

Layout, 4×4 tiles of 32×32:

    edge_tl   edge_t    edge_tr   inner_tl
    edge_l    center    edge_r    inner_tr
    edge_bl   edge_b    edge_br   inner_bl
    edge_t_b  center_b  edge_b_b  inner_br

"Inner" ground is the patch, "outer" surrounds it. Every boundary is a
wavy profile: the depth of outer ground measured in from one side, a
function of the position along that side. Each profile is periodic over
32 px and equals DEPTH at the tile border, so every edge self-tiles, every
corner meets its edges, and inner corners meet the edges beside them.
Tiles are classified per pixel from these profiles, evaluated a pixel
beyond the tile too, so the lip at a tile border matches its neighbor.

Light comes from the top-left and the outer ground (grass) stands above
the inner ground: grass edges that face down or right are in shade and
cast a lip of shadow onto the ground below and to the right of them;
grass edges that face up or left catch the light.
"""

import math

T = 32
DEPTH = 9  # outer ground depth at every tile border
CORNER = 5  # radius of rounded outer corners
NOTCH = 3  # radius of the rounded tip of an inner corner


def wave(*terms):
    """A periodic depth profile: DEPTH plus sine harmonics (zero at 0)."""
    out = []
    for x in range(T):
        v = sum(a * math.sin(2 * math.pi * k * x / T) for k, a in terms)
        out.append(DEPTH + round(v))
    return out


PROFILES = {
    "top": wave((1, 1.4), (3, 0.9)),
    "top_b": wave((1, -1.2), (2, 1.0), (5, 0.4)),
    "bottom": wave((1, -1.3), (2, -0.8)),
    "bottom_b": wave((1, 1.1), (3, -1.0)),
    "left": wave((1, 1.2), (2, -0.9)),
    "right": wave((1, -1.0), (3, 1.1)),
}


def depth(name, i):
    return PROFILES[name][i % T]


def rounded(x, y, cx, cy, r, quadrant):
    """False for points in a corner square cut off by a circle of radius r
    centered at (cx, cy); quadrant (sx, sy) is which side is cut."""
    sx, sy = quadrant
    if (x - cx) * sx > 0 and (y - cy) * sy > 0:
        return (x - cx) ** 2 + (y - cy) ** 2 <= r * r
    return True


def inner_fn(kind):
    """Is pixel (x, y) inner ground, for one tile kind? x and y may be
    one pixel outside the tile; the answer then matches the neighbor."""
    top = "top_b" if kind == "edge_t_b" else "top"
    bottom = "bottom_b" if kind == "edge_b_b" else "bottom"

    def t(x, y):
        return y >= depth(top, x)

    def b(x, y):
        return y <= T - 1 - depth(bottom, x)

    def lft(x, y):
        return x >= depth("left", y)

    def rgt(x, y):
        return x <= T - 1 - depth("right", y)

    c0, c1 = DEPTH + CORNER - 0.5, T - 1 - DEPTH - CORNER + 0.5
    n0, n1 = DEPTH - NOTCH - 0.5, T - 1 - DEPTH + NOTCH + 0.5
    table = {
        "center": lambda x, y: True,
        "center_b": lambda x, y: True,
        "edge_t": t,
        "edge_t_b": t,
        "edge_b": b,
        "edge_b_b": b,
        "edge_l": lft,
        "edge_r": rgt,
        "edge_tl": lambda x, y: t(x, y) and lft(x, y) and rounded(x, y, c0, c0, CORNER, (-1, -1)),
        "edge_tr": lambda x, y: t(x, y) and rgt(x, y) and rounded(x, y, c1, c0, CORNER, (1, -1)),
        "edge_bl": lambda x, y: b(x, y) and lft(x, y) and rounded(x, y, c0, c1, CORNER, (-1, 1)),
        "edge_br": lambda x, y: b(x, y) and rgt(x, y) and rounded(x, y, c1, c1, CORNER, (1, 1)),
        # Inner corners: outer ground only in one corner, a rounded notch.
        "inner_tl": lambda x, y: not (
            not t(x, y) and not lft(x, y) and rounded(x, y, n0, n0, NOTCH, (1, 1))
        ),
        "inner_tr": lambda x, y: not (
            not t(x, y) and not rgt(x, y) and rounded(x, y, n1, n0, NOTCH, (-1, 1))
        ),
        "inner_bl": lambda x, y: not (
            not b(x, y) and not lft(x, y) and rounded(x, y, n0, n1, NOTCH, (1, -1))
        ),
        "inner_br": lambda x, y: not (
            not b(x, y) and not rgt(x, y) and rounded(x, y, n1, n1, NOTCH, (-1, -1))
        ),
    }
    return table[kind]


ORDER = [
    "edge_tl", "edge_t", "edge_tr", "inner_tl",
    "edge_l", "center", "edge_r", "inner_tr",
    "edge_bl", "edge_b", "edge_br", "inner_bl",
    "edge_t_b", "center_b", "edge_b_b", "inner_br",
]  # fmt: skip


def tile(ctx, kind, outer, inner, colors):
    """One transition tile from an outer and an inner ground tile.

    colors: shade (outer edge facing down/right), lit (outer edge facing
    up/left), lip (inner ground just below/right of outer), lip2 (the
    second, softer row of the lip under a top edge).
    """
    is_inner = inner_fn(kind)
    out = ctx.canvas(T, T)
    for y in range(T):
        for x in range(T):
            if is_inner(x, y):
                c = inner.get(x, y)
                if not is_inner(x, y - 1) or not is_inner(x - 1, y):
                    c = colors["lip"]
                elif not is_inner(x, y - 2) and colors.get("lip2"):
                    c = colors["lip2"]
            else:
                c = outer.get(x, y)
                if is_inner(x, y + 1) or is_inner(x + 1, y):
                    c = colors["shade"]
                elif is_inner(x, y - 1) or is_inner(x - 1, y):
                    c = colors["lit"]
            out.px(x, y, c)
    return out


def tileset(ctx, outer, inner, inner_b, colors):
    return [tile(ctx, k, outer, inner_b if k == "center_b" else inner, colors) for k in ORDER]
