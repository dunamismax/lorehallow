"""The 404 page art, 160×120: lost in the woods. The hand-drawn hero stands
where a dirt path runs out under a wall of dark trees, puzzled, next to a
signpost whose two blank boards point opposite ways and whose third board
has broken off. Built from the game's own tiles, trees, rock and hero, so
it matches the world exactly.
"""

T = 32
COLS, ROWS = 6, 5
VIEW = (16, 0)  # top-left of the 160×120 window on the 192×160 map

# The path comes up from the bottom and stops in the middle of the clearing.
DIRT = {(2, 2), (3, 2), (2, 3), (3, 3), (2, 4), (3, 4)}


def in_dirt(c, r):
    if r >= ROWS:
        return c in (2, 3)
    return (c, r) in DIRT


def draw(ctx):
    ref = ctx.module("../style/reference.py")
    rng = ctx.rng("ground")
    m = ctx.canvas(COLS * T, ROWS * T)
    for r in range(ROWS):
        for c in range(COLS):
            name = "grass_a" if rng.random() < 0.5 else rng.choice(["grass_b", "grass_c", "grass_tuft"])
            if (c, r) in {(0, 3), (5, 2)}:
                name = "grass_flowers"
            m.stamp(ctx.frame("terrain_base", name), c * T, r * T)
            if in_dirt(c, r):
                m.stamp(ctx.frame("terrain_grass_dirt", ref.pick(in_dirt, c, r, rng)), c * T, r * T)

    # The woods close in: deep shade on the ground under the trees, deepest
    # at the top, with a ragged edge.
    shade = {"moss": "moss shadow", "leaf": "moss", "sunlit leaf": "leaf", "moss shadow": "deep forest",
             "parchment white": "sand", "lantern gold": "amber", "dry path": "earth", "sand": "dry path",
             "earth": "bark"}  # fmt: skip
    for y in range(m.h):
        for x in range(m.w):
            edge = 58 + (8 if (x // 11) % 3 == 0 else 0) + (5 if (x // 6) % 4 == 1 else 0)
            if y < edge and m.get(x, y) in shade:
                m.px(x, y, shade[m.get(x, y)])

    tree = ctx.art("prop_tree")
    hero = ctx.art("concept_hero")
    vx, vy = VIEW
    placed = [  # map coordinates, back to front
        (tree.mirror(), 30, -40),  # the wall of woods behind the clearing
        (tree, 112, -42),
        (tree, -14, -34),
        (tree.mirror(), 70, -30),
        (tree, 150, -32),
        (tree, -36, 10),  # framing trees at the sides
        (tree.mirror(), 166, 16),
        (ctx.pxl("../props/signpost.pxl.yaml")[0], 110, 62),
        (ctx.art("prop_mossy_rock"), 30, 88),
        (hero, 66, 52),
        (ctx.pxl("question.pxl.yaml")[0], 63, 58),
        (ref.bush(ctx, 36, 26), 146, 100),
        (ref.bush(ctx, 26, 20), 12, 108),
    ]
    for img, x, y in placed:
        m.stamp(img, x, y)

    # A few fireflies in the dark under the trees.
    for x, y in ((50, 50), (98, 40), (138, 56), (168, 44), (24, 64)):
        m.px(x, y, "glow")
    return m.crop(vx, vy, 160, 120)
