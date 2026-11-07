"""The left end of a timber-framed cottage, 80×96, for scenes framed by a
cottage at their right edge (the right side continues out of frame).

Top-down 3/4 view: the red tile roof on top, then the front wall: a
plastered, timber-framed upper storey with a lit window over a stone
ground storey with an arched plank door, stone steps, and ivy climbing
the left corner. Lit from the top-left, ink outline.
"""

W, H = 80, 96
ROOF_BOTTOM = 34  # last row of roof tiles
EAVE = (35, 37)  # fascia board rows
STONE_TOP = 58  # first row of the stone storey
DOOR = (44, 64)  # door columns [x0, x1)
FLOOR = 89  # last row of wall


def roof(c):
    # Rows of overlapping clay tiles; the roof's left end slopes inward
    # going up (a hipped end seen from above).
    for row, y0 in enumerate(range(0, ROOF_BOTTOM + 1, 5)):
        left = 14 - (y0 * 12) // ROOF_BOTTOM
        shift = 0 if row % 2 else 3
        for y in range(y0, min(y0 + 5, ROOF_BOTTOM + 1)):
            for x in range(left, W):
                k = (x - shift) % 6
                if y == y0 + 4:
                    col = "night plum" if k == 5 else "dusk violet-grey"
                elif k == 5:
                    col = "banner red" if y < y0 + 2 else "dusk violet-grey"
                elif y == y0 or k == 0:
                    col = "berry"
                else:
                    col = "banner red"
                c.px(x, y, col)
        # the hip edge catches the light
        c.px(left, y0, "dry path")
        c.px(left, y0 + 1, "berry")
    # ridge cap along the top
    c.rect(14, 0, W - 14, 1, "ember")
    # fascia board under the eaves
    c.rect(2, EAVE[0], W - 2, 1, "dry path")
    c.rect(2, EAVE[0] + 1, W - 2, 1, "earth")
    c.rect(2, EAVE[1], W - 2, 1, "bark")


def upper_wall(c):
    for y in range(EAVE[1] + 1, STONE_TOP):
        for x in range(3, W):
            c.px(x, y, "sand" if x > 4 else "parchment white")
    # the eave's shadow on the plaster
    c.rect(3, EAVE[1] + 1, W - 3, 2, "pale stone")
    # timber frame: posts, a rail and braces, lit on their left
    for x in (3, 34, 66):
        c.rect(x, EAVE[1] + 1, 3, STONE_TOP - EAVE[1] - 1, "bark")
        c.line(x, EAVE[1] + 1, x, STONE_TOP - 1, "earth")
    c.rect(3, STONE_TOP - 3, W - 3, 3, "bark")
    c.rect(3, STONE_TOP - 3, W - 3, 1, "earth")
    for k in range(12):
        c.px(37 + k, 52 - k * 13 // 12, "bark")
        c.px(65 - k, 52 - k * 13 // 12, "bark")
    # a lit window with four panes and a sill
    x0, y0 = 13, 42
    c.rect(x0, y0, 13, 11, "ink")
    c.rect(x0 + 1, y0 + 1, 11, 9, "lantern gold")
    c.rect(x0 + 1, y0 + 1, 4, 3, "glow")
    c.rect(x0 + 7, y0 + 1, 3, 2, "glow")
    c.rect(x0 + 1, y0 + 6, 5, 1, "amber")
    c.rect(x0 + 6, y0 + 1, 1, 9, "bark")
    c.rect(x0 + 1, y0 + 5, 11, 1, "bark")
    c.rect(x0 + 10, y0 + 6, 2, 4, "amber")
    c.rect(x0 - 1, y0 + 11, 15, 1, "dry path")
    c.rect(x0 - 1, y0 + 12, 15, 1, "bark")


def stone_storey(c):
    # coursed stone blocks with slate mortar
    for y in range(STONE_TOP, FLOOR + 1):
        for x in range(2, W):
            c.px(x, y, "slate")
    courses = [(STONE_TOP, 6, 0), (STONE_TOP + 6, 5, 4), (STONE_TOP + 11, 6, 2),
               (STONE_TOP + 17, 5, 6), (STONE_TOP + 22, 5, 1), (STONE_TOP + 27, 5, 5)]  # fmt: skip
    for y0, h, off in courses:
        x = 2 - off
        widths = (9, 7, 10, 8, 6, 9, 11, 7, 8, 10, 9, 7)
        for k, w in enumerate(widths):
            if x >= W:
                break
            block(c, max(x, 2), y0, min(x + w - 1, W) - max(x, 2), h - 1, k)
            x += w
    # arched doorway in a pale stone surround
    x0, x1 = DOOR
    cx = (x0 + x1 - 1) / 2
    r = (x1 - x0) / 2 + 2
    top = STONE_TOP - 2
    for y in range(top, FLOOR + 1):
        for x in range(x0 - 2, x1 + 2):
            dy = max(0.0, (top + r) - y)
            if (x - cx) ** 2 + dy**2 <= r * r:
                c.px(x, y, "pale stone" if (x + y // 3) % 5 else "stone")
    r -= 2
    for y in range(top + 2, FLOOR + 1):
        for x in range(x0, x1):
            dy = max(0.0, (top + 2 + r) - y)
            if (x - cx) ** 2 + dy**2 <= r * r:
                plank = (x - x0) % 5
                col = "dark soil" if plank == 4 else "earth" if plank == 0 else "bark"
                c.px(x, y, col)
    # the doorway's top-left inner edge is in shadow (a recess)
    for y in range(top + 2, FLOOR + 1):
        for x in range(x0, x1):
            if c.get(x, y) in ("earth", "bark") and (
                c.get(x, y - 1) in ("pale stone", "stone") or c.get(x - 1, y) in ("pale stone", "stone")
            ):
                c.px(x, y, "dark soil")
    # iron bands and a ring handle
    for y in (STONE_TOP + 8, STONE_TOP + 22):
        c.rect(x0 + 1, y, x1 - x0 - 2, 1, "night plum")
    c.points([(x1 - 6, STONE_TOP + 15), (x1 - 5, STONE_TOP + 14), (x1 - 4, STONE_TOP + 15),
              (x1 - 5, STONE_TOP + 16)], "lantern gold")  # fmt: skip


def block(c, x, y, w, h, k):
    if w <= 0:
        return
    body = "stone" if k % 3 else "pale stone"
    c.rect(x, y, w, h, body)
    c.rect(x, y, w, 1, "pale stone")
    c.rect(x, y, 1, h, "pale stone")
    c.rect(x + 1, y + h - 1, w - 1, 1, "dusk violet-grey" if body == "stone" else "stone")


def steps(c):
    x0, x1 = DOOR[0] - 4, DOOR[1] + 4
    c.rect(x0, FLOOR + 1, x1 - x0, 2, "pale stone")
    c.rect(x0, FLOOR + 3, x1 - x0, 1, "stone")
    c.rect(x0 - 2, FLOOR + 4, x1 - x0 + 4, 1, "pale stone")
    c.rect(x0 - 2, FLOOR + 5, x1 - x0 + 4, 1, "slate")
    for x in range(x0 + 5, x1, 9):
        c.px(x, FLOOR + 2, "stone")


def ivy(c, rng):
    stems = [(4, FLOOR), (8, FLOOR - 2)]
    for sx, sy in stems:
        x, y = sx, sy
        for _ in range(26):
            c.px(x, y, "moss shadow")
            if rng.random() < 0.55:
                lx = x + rng.choice((-1, 1))
                c.points([(lx, y), (lx, y - 1)], "leaf")
                c.px(lx + (1 if lx > x else -1), y - 1, "moss")
            y -= 1
            x += rng.choice((0, 0, 1, -1))
            x = max(2, min(x, 12))
    c.points([(3, 70), (4, 69), (2, 75), (9, 66), (10, 65)], "sunlit leaf")


def draw(ctx):
    c = ctx.canvas(W, H)
    roof(c)
    upper_wall(c)
    stone_storey(c)
    steps(c)
    ivy(c, ctx.rng("ivy"))
    c.outline("ink")
    return c
