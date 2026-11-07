"""Base terrain: 16 ground tiles, 32×32, in the manifest's frame order.

Every tile is drawn on a wrap-around canvas, so it is seamless with
itself. The variants of a material share one layout: features that reach
within BAND px of the tile border are identical in every variant, and
only features wholly inside change, so all variants tile with each other.
Ground tiles have no outline and stay low in contrast (STYLE.md "Tiles").

Textures are deliberate clusters, not noise: lit patches grown from a
seed pixel with a darker rim on their bottom-right, grass tufts, and for
paved ground a wrap-around cell layout with 1 px joints, each stone lit
on its top-left and shaded on its bottom-right.
"""

T = 32
BAND = 3  # features reaching this close to the border are shared by every variant


# -- clusters -----------------------------------------------------------------


def grow(rng, x, y, n, wide=True):
    """A compact cluster of n pixels grown from (x, y), wider than tall."""
    pts = [(x, y)]
    seen = {(x, y)}
    while len(pts) < n:
        bx, by = rng.choice(pts)
        dx, dy = rng.choice([(1, 0), (-1, 0), (1, 0), (-1, 0), (0, 1), (0, -1)] if wide else
                            [(1, 0), (-1, 0), (0, 1), (0, -1)])  # fmt: skip
        p = (bx + dx, by + dy)
        if p not in seen:
            seen.add(p)
            pts.append(p)
    return pts


def touches_border(pts):
    return any(
        x % T < BAND or x % T >= T - BAND or y % T < BAND or y % T >= T - BAND for x, y in pts
    )


def patch(t, pts, color, rim=None):
    """Fill a cluster; give it a shadow rim along its underside so it
    reads as a lit, slightly raised clump."""
    s = {(x % T, y % T) for x, y in pts}
    for x, y in s:
        t.px(x, y, color)
    if rim:
        for x, y in s:
            q = (x, (y + 1) % T)
            if q not in s:
                t.px(*q, rim)


# Grass tufts: three or four blades meeting at a root.
TUFTS = [
    ["..L..", "L.L.L", ".LLL.", "..S.."],
    ["L...L", ".L.L.", ".LLL.", "..SS."],
    [".L.L", "L.LL", ".LLS"],
    ["L.L.", ".LL.", ".LLS"],
]
TALL_TUFT = [
    "....H.......",
    ".H..L..H....",
    ".L..L.HL..H.",
    "..L.LL.L.HL.",
    "..LL.LLL.L..",
    "...LLLLLLL..",
    "...SLLLLLSS.",
    "....SSSSS...",
]
COLORS = {"L": "leaf", "H": "sunlit leaf", "S": "moss shadow"}


def stamp_rows(t, x, y, rows):
    for j, row in enumerate(rows):
        for i, ch in enumerate(row):
            if ch != ".":
                t.px(x + i, y + j, COLORS[ch])


def rows_pts(x, y, rows):
    return [(x + i, y + j) for j, row in enumerate(rows) for i, ch in enumerate(row) if ch != "."]


def jitter_grid(rng, cells, jitter, offset):
    step = T // cells
    return [
        (offset[0] + gx * step + rng.randrange(jitter), offset[1] + gy * step + rng.randrange(jitter))
        for gy in range(cells)
        for gx in range(cells)
    ]


class Texture:
    """A material: a base color plus a layout of feature-drawing calls.

    Each feature is (points, draw_fn). tile(rng) redraws the features that
    touch the border unchanged and re-rolls the rest with rng.
    """

    def __init__(self, ctx, base, make):
        self.ctx = ctx
        self.base = base
        self.make = make  # make(rng) -> list of (points, draw_fn)
        self.layout = make(ctx.rng(f"layout-{base}"))

    def tile(self, rng=None, extra=None):
        t = self.ctx.canvas(T, T, wrap=True)
        t.fill(self.base)
        fresh = self.make(rng) if rng else []
        kept = [f for f in self.layout if touches_border(f[0])] if rng else self.layout
        inner = [f for f in fresh if not touches_border(f[0])]
        for _pts, fn in kept + inner:
            fn(t)
        if extra:
            extra(t)
        return t


def grass_features(rng):
    out = []
    for x, y in jitter_grid(rng, 3, 7, (2, 3)):
        pts = grow(rng, x, y, rng.randrange(6, 11))
        out.append((pts, lambda t, p=pts: patch(t, p, "leaf")))
    for x, y in jitter_grid(rng, 4, 5, (1, 0)):
        rows = rng.choice(TUFTS)
        out.append((rows_pts(x, y, rows), lambda t, x=x, y=y, r=rows: stamp_rows(t, x, y, r)))
    for x, y in jitter_grid(rng, 3, 8, (6, 8)):
        pts = grow(rng, x, y, rng.randrange(2, 4))
        out.append((pts, lambda t, p=pts: patch(t, p, "moss shadow")))
    return out


def dirt_features(rng):
    out = []
    for x, y in jitter_grid(rng, 3, 6, (2, 2)):
        pts = grow(rng, x, y, rng.randrange(4, 9))
        out.append((pts, lambda t, p=pts: patch(t, p, "dry path", "bark")))
    for x, y in jitter_grid(rng, 3, 7, (7, 8)):
        pts = grow(rng, x, y, rng.randrange(1, 3))
        out.append((pts, lambda t, p=pts: patch(t, p, "bark")))
    return out


def stone_features(rng):
    out = []
    for x, y in jitter_grid(rng, 3, 6, (2, 3)):
        pts = grow(rng, x, y, rng.randrange(6, 12))
        out.append((pts, lambda t, p=pts: patch(t, p, "pale stone", "slate")))
    for x, y in jitter_grid(rng, 3, 7, (7, 9)):
        pts = grow(rng, x, y, rng.randrange(2, 5))
        out.append((pts, lambda t, p=pts: patch(t, p, "slate")))
    return out


# -- paving ------------------------------------------------------------------


def cells(points, metric):
    """Nearest-point cell id for every pixel on the 32×32 torus."""
    grid = [[0] * T for _ in range(T)]
    for y in range(T):
        for x in range(T):
            best, bi = None, 0
            for i, (px, py, w) in enumerate(points):
                dx = min(abs(x - px), T - abs(x - px))
                dy = min(abs(y - py), T - abs(y - py))
                d = metric(dx, dy) * w
                if best is None or d < best:
                    best, bi = d, i
            grid[y][x] = bi
    return grid


def pave(t, grid, joint, body, light, dark, tone=None):
    """Draw cells with 1 px joints on their right and bottom, cut corners,
    a lit top-left edge and a shaded bottom-right edge."""

    def cid(x, y):
        return grid[y % T][x % T]

    def is_joint(x, y):
        c = cid(x, y)
        return c != cid(x + 1, y) or c != cid(x, y + 1)

    J = [[is_joint(x, y) for x in range(T)] for y in range(T)]

    def j(x, y):
        return J[y % T][x % T]

    for y in range(T):
        for x in range(T):
            if j(x, y):
                t.px(x, y, joint)
                continue
            up, left, down, right = j(x, y - 1), j(x - 1, y), j(x, y + 1), j(x + 1, y)
            if (up and left) or (down and right) or (up and right) or (down and left):
                t.px(x, y, joint)  # round the corners
                continue
            c = (tone or {}).get(cid(x, y), body)
            if up or left:
                c = light
            elif down or right:
                c = dark
            t.px(x, y, c)


def cobble_points(rng):
    pts = []
    for row in range(5):
        y = row * 6.4 + 3
        shift = 3.2 if row % 2 else 0
        for col in range(5):
            x = col * 6.4 + shift + rng.uniform(-1, 1)
            pts.append(((x % T), (y + rng.uniform(-0.8, 0.8)) % T, rng.uniform(0.9, 1.1)))
    return pts


def euclid(dx, dy):
    return (dx * dx + dy * dy) ** 0.5


def blocky(dx, dy):
    return max(dx, dy) * 0.75 + (dx + dy) * 0.25


def border_band(grid):
    return [grid[y][x] for y in range(T) for x in range(T) if min(x, y, T - 1 - x, T - 1 - y) < BAND]


def variant_points(points, rng, metric, moves=12):
    """Move interior points while the border band of the layout stays the
    same, so the variant still tiles with the original."""
    base = border_band(cells(points, metric))
    pts = list(points)
    for _ in range(moves):
        i = rng.randrange(len(pts))
        x, y, w = pts[i]
        trial = list(pts)
        trial[i] = (x + rng.uniform(-1.6, 1.6), y + rng.uniform(-1.6, 1.6), w * rng.uniform(0.9, 1.1))
        if border_band(cells(trial, metric)) == base:
            pts = trial
    return pts


def cobbles(ctx, points, rng_tone):
    t = ctx.canvas(T, T, wrap=True)
    grid = cells(points, euclid)
    tone = {i: "stone" for i in range(len(points))}
    for i in range(len(points)):
        if rng_tone.random() < 0.18:
            tone[i] = "slate"
    pave(t, grid, "dusk violet-grey", "stone", "pale stone", "slate", tone)
    return t, grid


FLAG_POINTS = [
    (5, 6, 1.0), (20, 4, 1.15), (29, 13, 0.95), (12, 15, 1.05),
    (3, 23, 1.0), (19, 25, 1.1), (27, 28, 0.9),
]  # fmt: skip


def flagstones(ctx):
    t = ctx.canvas(T, T, wrap=True)
    grid = cells(FLAG_POINTS, blocky)
    pave(t, grid, "slate", "pale stone", "pale stone", "stone")
    for x, y in ((9, 4), (24, 9), (14, 18), (6, 26), (22, 22)):
        t.points([(x, y), (x + 1, y)], "stone")
    t.px(16, 11, "sand")
    t.px(25, 29, "sand")
    return t


# -- details -------------------------------------------------------------------


def flower(t, x, y, petal="parchment white", heart="lantern gold"):
    t.points([(x, y - 1), (x - 1, y), (x + 1, y)], petal)
    t.px(x, y, heart)
    t.px(x + 1, y + 1, "moss shadow")


def pebble(t, x, y, w=3, under="moss shadow"):
    """A small stone sitting in the ground, with a contact shadow."""
    t.rect(x, y, w, 1, "pale stone")
    t.rect(x - 1, y + 1, w + 1, 1, "stone")
    t.px(x - 1, y + 1, "pale stone")
    t.rect(x, y + 2, w, 1, "slate")
    t.px(x + w, y + 1, "slate")
    t.rect(x + 1, y + 3, w, 1, under)
    t.px(x + w + 1, y + 2, under)


def crack(t, pts):
    for (x0, y0), (x1, y1) in zip(pts, pts[1:], strict=False):
        t.line(x0, y0, x1, y1, "dusk violet-grey")
    for x, y in pts[1:-1]:
        t.px(x, y + 1, "pale stone")


def moss_patch(t, rng, x, y, n):
    pts = grow(rng, x, y, n)
    patch(t, pts, "moss", "moss shadow")
    for px, py in pts[:2]:
        t.px(px, py, "leaf")


def draw(ctx):
    v = ctx.rng("variants")
    grass = Texture(ctx, "moss", grass_features)
    dirt = Texture(ctx, "earth", dirt_features)
    stone = Texture(ctx, "stone", stone_features)

    def flowers(t):
        flower(t, 10, 11)
        flower(t, 21, 9, "lantern gold", "amber")
        flower(t, 14, 21)
        flower(t, 23, 20)

    def tall(t):
        stamp_rows(t, 10, 13, TALL_TUFT)

    def pebbles(t):
        pebble(t, 9, 9, 3, "bark")
        pebble(t, 20, 16, 2, "bark")
        pebble(t, 12, 21, 3, "bark")

    def stones_in_grass(t):
        pebble(t, 9, 10, 4)
        pebble(t, 21, 17)
        pebble(t, 12, 21, 2)

    def cracked(t):
        crack(t, [(7, 8), (11, 11), (14, 11), (17, 15), (17, 19), (21, 23), (24, 23)])
        crack(t, [(17, 15), (21, 14)])

    def mossy(t):
        r = ctx.rng("moss")
        moss_patch(t, r, 9, 10, 9)
        moss_patch(t, r, 19, 19, 12)
        moss_patch(t, r, 21, 8, 4)

    cpts = cobble_points(ctx.rng("cobble"))
    cobble_a, _ = cobbles(ctx, cpts, ctx.rng("tone_a"))
    cobble_b, _ = cobbles(ctx, variant_points(cpts, ctx.rng("cobble_b"), euclid), ctx.rng("tone_b"))

    return [
        grass.tile(),
        grass.tile(v),
        grass.tile(v),
        grass.tile(v, flowers),
        grass.tile(v, tall),
        dirt.tile(),
        dirt.tile(v),
        dirt.tile(v, pebbles),
        stone.tile(),
        stone.tile(v),
        stone.tile(v, cracked),
        stone.tile(v, mossy),
        cobble_a,
        cobble_b,
        flagstones(ctx),
        grass.tile(v, stones_in_grass),
    ]
