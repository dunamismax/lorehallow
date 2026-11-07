"""The style reference vignette, 240×160: a composite of the hand-drawn
tiles, hero and props, laid out like the original vignette: a big oak on
the left with woods behind, a mossy rock, a lantern post, a cottage on
the right, flagstones, a dirt path with a branch to the cottage door,
flowers in the grass, and the hero on the path.

The ground is a real tile map, autotiled from the transition sets the
way the game will, so the vignette also shows that the tiles fit.
"""

T = 32
COLS, ROWS = 9, 6
VIEW = (16, 20)  # top-left of the 240×160 window on the 288×192 map

# Patches: (material, set of (col, row)); cells off the map count as
# inner where the patch runs out of frame.
DIRT = {(c, r) for r in (4, 5) for c in range(COLS)} | {(6, 2), (7, 2), (6, 3), (7, 3)}
STONE = {(3, 2), (4, 2), (3, 3), (4, 3)}
ORDER = [
    "edge_tl", "edge_t", "edge_tr", "inner_tl",
    "edge_l", "center", "edge_r", "inner_tr",
    "edge_bl", "edge_b", "edge_br", "inner_bl",
    "edge_t_b", "center_b", "edge_b_b", "inner_br",
]  # fmt: skip
GRASS = ["grass_a", "grass_b", "grass_c", "grass_flowers", "grass_tuft", "grass_stones"]


def in_dirt(c, r):
    if r >= ROWS:
        return True
    if c < 0 or c >= COLS:
        return r in (4, 5)
    return (c, r) in DIRT


def in_stone(c, r):
    return (c, r) in STONE


def pick(inside, c, r, rng):
    o = lambda dc, dr: not inside(c + dc, r + dr)  # noqa: E731
    t, b, lf, rt = o(0, -1), o(0, 1), o(-1, 0), o(1, 0)
    if t and lf:
        return "edge_tl"
    if t and rt:
        return "edge_tr"
    if b and lf:
        return "edge_bl"
    if b and rt:
        return "edge_br"
    if t:
        return rng.choice(["edge_t", "edge_t", "edge_t_b"])
    if b:
        return rng.choice(["edge_b", "edge_b_b"])
    if lf:
        return "edge_l"
    if rt:
        return "edge_r"
    for name, (dc, dr) in (("inner_tl", (-1, -1)), ("inner_tr", (1, -1)),
                           ("inner_bl", (-1, 1)), ("inner_br", (1, 1))):  # fmt: skip
        if o(dc, dr):
            return name
    return rng.choice(["center", "center", "center_b"])


def ground(ctx):
    rng = ctx.rng("ground")
    m = ctx.canvas(COLS * T, ROWS * T)
    # grass: mostly plain, with flowers where the old vignette had them
    flowers = {(1, 3), (8, 3), (5, 1), (2, 1)}
    for r in range(ROWS):
        for c in range(COLS):
            if (c, r) in flowers:
                name = "grass_flowers"
            else:
                name = "grass_a" if rng.random() < 0.5 else rng.choice(GRASS[1:3] + ["grass_tuft"])
            m.stamp(ctx.frame("terrain_base", name), c * T, r * T)
            if in_dirt(c, r):
                m.stamp(ctx.frame("terrain_grass_dirt", pick(in_dirt, c, r, rng)), c * T, r * T)
            elif in_stone(c, r):
                m.stamp(ctx.frame("terrain_grass_stone", pick(in_stone, c, r, rng)), c * T, r * T)
    return m


def bush(ctx, w, h):
    sh = ctx.module("../props/shading.py")
    b = ctx.canvas(w, h)
    for cx, cy, rx, ry in ((w * 0.3, h * 0.55, w * 0.3, h * 0.4), (w * 0.7, h * 0.5, w * 0.32, h * 0.45),
                           (w * 0.5, h * 0.4, w * 0.3, h * 0.38), (w * 0.45, h * 0.7, w * 0.35, h * 0.28)):  # fmt: skip
        sh.clump(b, cx, cy, rx, ry, sh.FOLIAGE, [0.18, 0.42, 0.66, 0.86])
    sh.bumps(b, ctx.rng(f"bush{w}"), sh.FOLIAGE, 6)
    sh.prune(b)
    b.outline("ink")
    return b


def draw(ctx):
    m = ground(ctx)
    tree = ctx.art("prop_tree")
    vx, vy = VIEW
    # Props, back to front (by the row they stand on), in map coordinates.
    placed = [
        (tree, 150, -42),  # woods behind
        (tree.mirror(), 104, -36),
        (tree.mirror(), 46, -30),
        (tree, 76, -14),
        (ctx.art("prop_cottage_corner"), 176, 4),
        (ctx.art("prop_lantern_post"), 140, 36),
        (tree, 4, 4),  # the big oak
        (ctx.art("prop_mossy_rock"), 74, 70),
        (ctx.art("concept_hero"), 116, 104),
        (bush(ctx, 40, 30), 228, 150),
        (bush(ctx, 28, 22), 14, 166),
    ]
    # deep shade on the ground under the woods along the top
    shade = {"moss": "moss shadow", "leaf": "moss", "sunlit leaf": "leaf",
             "moss shadow": "deep forest", "parchment white": "sand", "lantern gold": "amber"}  # fmt: skip
    for y in range(0, 64):
        for x in range(m.w):
            edge = 44 + (7 if (x // 9) % 3 == 0 else 0) + (4 if (x // 5) % 4 == 1 else 0)
            if y < edge and m.get(x, y) in shade:
                m.px(x, y, shade[m.get(x, y)])
    for img, x, y in placed:
        m.stamp(img, x, y)
    return m.crop(vx, vy, 240, 160)
