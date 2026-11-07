"""The 20 progress-map landmarks, 32×32 each, one per BUILD.md phase, from
a campfire to a castle (manifest order). Most stand on the same small
grass mound so the map reads as one journey. Ink outline, light from the
top-left, flat shading steps.

Each landmark is drawn on its own canvas from a few shared building
blocks (mound, box, gable roof, flag, form-lit clumps), then outlined.
"""

GRASS = ["moss shadow", "moss", "leaf", "sunlit leaf"]
WOOD = ("dry path", "earth", "bark")
STONE = ("pale stone", "stone", "slate")
PLASTER = ("parchment white", "sand", "pale stone")
ROOF = ("berry", "banner red", "dusk violet-grey")
SLATE_ROOF = ("stone", "slate", "dusk violet-grey")
WATER = ["deep dusk", "dusk blue", "river", "sky"]


class Kit:
    def __init__(self, ctx):
        self.ctx = ctx
        self.sh = ctx.module("../props/shading.py")
        self.lantern = ctx.pxl("../props/lantern.pxl.yaml")[0]

    def new(self):
        return self.ctx.canvas(32, 32)

    def mound(self, c, w=26, top=25):
        self.sh.clump(c, 15.5, top + 2.5, w / 2, 3.5, GRASS, [0.25, 0.5, 0.75])

    def clump(self, c, cx, cy, rx, ry, ramp, cuts=(0.2, 0.45, 0.75)):
        self.sh.clump(c, cx, cy, rx, ry, ramp, list(cuts))

    def finish(self, c):
        self.sh.prune(c, 1)
        c.outline("ink")
        return c


def box(c, x, y, w, h, ramp):
    """A lit block: top and left edges light, bottom and right edges dark."""
    light, mid, dark = ramp
    c.rect(x, y, w, h, mid)
    c.rect(x, y, w, 1, light)
    c.rect(x, y, 1, h, light)
    c.rect(x + w - 1, y + 1, 1, h - 1, dark)
    c.rect(x + 1, y + h - 1, w - 1, 1, dark)


def gable(c, x, y, w, h, ramp):
    """A gable roof w wide, apex h rows above y + h; left slope lit."""
    light, mid, dark = ramp
    for j in range(h):
        half = (w * (j + 1)) // (2 * h)
        cx = x + w // 2
        for i in range(cx - half, cx + half):
            c.px(i, y + j, light if i < cx else mid)
    c.rect(x, y + h - 1, w, 1, dark)


def flag(c, x, y, colors=("berry", "banner red"), length=5):
    c.line(x, y, x, y + 7, "bark")
    for j in range(3):
        c.rect(x + 1, y + j, length - j, 1, colors[0] if j == 0 else colors[1])


def window(c, x, y, w=2, h=2):
    c.rect(x, y, w, h, "lantern gold")
    c.px(x, y, "glow")


# -- the landmarks ---------------------------------------------------------------


FLAME = [
    "......a.....",
    ".....aa..a..",
    "....aga..aa.",
    "...aggaa.aga",
    "...agGgaagga",
    "..agGGGgagGa",
    "..agGwGGgGGa",
    ".rgGwwGGGgar",
    ".ragGwwGGgar",
    "..ragGGGgar.",
    "...rraaarr..",
]
FLAME_COLORS = {"r": "ember", "a": "amber", "g": "lantern gold", "G": "glow", "w": "parchment white"}


def campfire(k):
    c = k.new()
    k.mound(c)
    for x0, y0, x1, y1 in ((8, 26, 22, 22), (9, 22, 23, 26)):
        c.line(x0, y0, x1, y1, "bark")
        c.line(x0, y0 - 1, x1, y1 - 1, "earth")
    for x in (7, 12, 18, 23):
        k.clump(c, x + 1, 27, 2, 1.6, ["slate", "stone", "pale stone"])
    for j, row in enumerate(FLAME):
        for i, ch in enumerate(row):
            if ch != ".":
                c.px(10 + i, 10 + j, FLAME_COLORS[ch])
    c.points([(12, 7), (20, 8)], "amber")
    return k.finish(c)


def signpost(k):
    c = k.new()
    k.mound(c)
    box(c, 14, 6, 3, 21, WOOD)
    for y, x0, d in ((8, 6, 1), (14, 13, -1)):
        box(c, x0, y, 13, 4, WOOD)
        tip = x0 + 12 if d > 0 else x0
        c.px(tip + d, y + 1, "earth")
        c.px(tip + d, y + 2, "bark")
    c.rect(8, 9, 7, 1, "bark")
    c.rect(17, 15, 6, 1, "bark")
    return k.finish(c)


def crossroads(k):
    c = k.new()
    k.mound(c, 30, 21)
    k.sh.clump(c, 15.5, 24, 15, 6, GRASS, [0.25, 0.5, 0.75])
    for y in range(19, 30):
        for x in range(1, 31):
            if c.opaque(x, y) and (abs((x - 15.5) - 2.2 * (y - 24)) < 2.2 or abs(y - 24) < 1.2):
                c.px(x, y, "dry path" if (x + y) % 7 else "sand")
    box(c, 22, 12, 3, 11, WOOD)
    box(c, 19, 13, 9, 3, WOOD)
    c.px(27, 14, "bark")
    k.clump(c, 7, 19, 3, 2.5, ["slate", "stone", "pale stone"])
    return k.finish(c)


def market_stall(k):
    c = k.new()
    k.mound(c)
    box(c, 5, 10, 2, 16, WOOD)
    box(c, 25, 10, 2, 16, WOOD)
    for i in range(22):
        col = "berry" if (i // 4) % 2 == 0 else "parchment white"
        c.line(5 + i, 7, 5 + i, 11, col)
    c.rect(5, 11, 22, 1, "banner red")
    for i in range(0, 22, 4):
        c.px(6 + i, 12, "berry")
        c.px(7 + i, 12, "berry")
    box(c, 6, 18, 20, 7, WOOD)
    k.clump(c, 10, 17, 3, 2, ["banner red", "berry", "berry", "sand"])
    k.clump(c, 16, 17, 3, 2, ["moss", "leaf", "sunlit leaf"])
    k.clump(c, 22, 17, 3, 2, ["ember", "amber", "lantern gold"])
    return k.finish(c)


def inn(k):
    c = k.new()
    k.mound(c)
    box(c, 6, 14, 20, 12, PLASTER)
    c.rect(6, 19, 20, 1, "bark")
    for x in (6, 15, 25):
        c.line(x, 14, x, 25, "bark")
    gable(c, 3, 3, 26, 11, ROOF)
    box(c, 13, 20, 5, 6, WOOD)
    c.px(16, 23, "lantern gold")
    window(c, 9, 16)
    window(c, 20, 16)
    window(c, 9, 21)
    c.rect(22, 6, 3, 4, "slate")
    c.px(22, 6, "stone")
    # hanging sign
    c.line(26, 15, 29, 15, "bark")
    box(c, 27, 16, 4, 3, ("lantern gold", "amber", "ember"))
    return k.finish(c)


def bell_tower(k):
    c = k.new()
    k.mound(c)
    box(c, 10, 11, 12, 16, STONE)
    for y in (15, 19, 23):
        c.rect(11, y, 10, 1, "slate")
    c.rect(13, 12, 6, 6, "night plum")
    k.clump(c, 15.5, 14.5, 2.5, 2.5, ["amber", "lantern gold", "glow"])
    c.rect(15, 17, 2, 1, "amber")
    gable(c, 8, 2, 16, 9, SLATE_ROOF)
    c.line(16, 0, 16, 2, "lantern gold")
    box(c, 13, 22, 6, 5, WOOD)
    return k.finish(c)


def training_dummies(k):
    c = k.new()
    k.mound(c)
    for x in (9, 22):
        box(c, x - 1, 17, 2, 10, WOOD)
        box(c, x - 6, 13, 12, 2, WOOD)
        k.clump(c, x, 15, 4, 4, ["dry path", "sand", "sand", "parchment white"])
        c.rect(x - 3, 16, 7, 1, "bark")
        k.clump(c, x, 8, 3, 3, ["dry path", "sand", "parchment white"])
    c.points([(21, 7), (23, 7), (22, 9)], "banner red")
    return k.finish(c)


def watchtower(k):
    c = k.new()
    k.mound(c)
    for x0, x1 in ((9, 12), (22, 19)):
        c.line(x0, 27, x1, 12, "earth")
        c.line(x0 + 1, 27, x1 + 1, 12, "bark")
    c.line(11, 20, 21, 20, "bark")
    c.line(11, 20, 20, 13, "bark")
    box(c, 7, 9, 18, 4, WOOD)
    for x in (8, 12, 16, 20, 23):
        c.line(x, 5, x, 9, "earth")
    gable(c, 6, 0, 20, 6, ROOF)
    c.px(15, 7, "lantern gold")
    return k.finish(c)


def treasure_chest(k):
    c = k.new()
    k.mound(c)
    box(c, 7, 16, 18, 10, WOOD)
    for x in (7, 15, 24):
        c.line(x, 16, x, 25, "amber")
    c.rect(7, 16, 18, 1, "lantern gold")
    # open lid tilted back, coins heaped and glowing
    box(c, 7, 9, 18, 5, WOOD)
    c.rect(7, 9, 18, 1, "lantern gold")
    k.clump(c, 16, 15, 8, 3, ["amber", "lantern gold", "glow", "parchment white"])
    c.points([(12, 13), (19, 12), (16, 12)], "glow")
    box(c, 14, 19, 4, 4, ("glow", "lantern gold", "amber"))
    c.px(15, 21, "ink")
    return k.finish(c)


def quest_board(k):
    c = k.new()
    k.mound(c)
    box(c, 6, 9, 2, 18, WOOD)
    box(c, 24, 9, 2, 18, WOOD)
    box(c, 5, 7, 22, 15, WOOD)
    c.rect(5, 6, 22, 1, "bark")
    for x, y, w, h in ((8, 9, 6, 5), (16, 10, 8, 6), (9, 15, 5, 5), (17, 17, 6, 3)):
        box(c, x, y, w, h, ("parchment white", "parchment white", "sand"))
        c.rect(x + 1, y + 2, w - 3, 1, "pale stone")
        c.px(x + w // 2, y, "berry")
    c.px(21, 12, "banner red")
    return k.finish(c)


def banner_camp(k):
    c = k.new()
    k.mound(c)
    for j in range(13):
        half = 2 + j
        for i in range(-half, half):
            c.px(14 + i, 13 + j, "parchment white" if i < 0 else "sand")
    c.rect(1, 25, 27, 1, "pale stone")
    for j in range(5):
        c.rect(14 - (j * 3) // 4, 21 + j, 2 + (j * 3) // 2, 1, "night plum")
    c.line(25, 4, 25, 25, "bark")
    c.line(24, 4, 24, 25, "earth")
    for j in range(7):
        c.rect(26, 5 + j, 5 - (j // 3), 1, "berry" if j == 0 else "banner red")
    c.points([(28, 7), (28, 8)], "lantern gold")
    return k.finish(c)


def bridge(k):
    c = k.new()
    k.sh.clump(c, 15.5, 25, 15, 5, WATER, [0.25, 0.5, 0.8])
    for y in range(13, 21):
        for x in range(2, 30):
            arch = ((x - 15.5) / 8) ** 2 + ((y - 21) / 6) ** 2 < 1
            if not arch:
                c.px(x, y, "stone" if (x // 4 + y // 3) % 2 else "pale stone")
    c.rect(2, 12, 28, 2, "pale stone")
    c.rect(2, 11, 28, 1, "stone")
    for x in range(3, 30, 5):
        box(c, x, 8, 3, 3, STONE)
    for y in range(15, 21):
        for x in range(8, 24):
            if ((x - 15.5) / 8) ** 2 + ((y - 21) / 6) ** 2 < 1:
                c.px(x, y, "night plum")
    c.points([(6, 26), (7, 26), (20, 27), (21, 27), (12, 24)], "mist")
    return k.finish(c)


def lighthouse(k):
    c = k.new()
    k.clump(c, 15.5, 27, 13, 4, ["slate", "stone", "pale stone"])
    for y in range(9, 26):
        half = 4 + (y - 9) // 6
        for x in range(16 - half, 16 + half):
            band = ((y - 9) // 4) % 2
            light = x < 16
            col = ("parchment white" if light else "pale stone") if band else (
                "berry" if light else "banner red")  # fmt: skip
            c.px(x, y, col)
    box(c, 11, 4, 10, 5, ("glow", "lantern gold", "amber"))
    c.rect(13, 5, 1, 3, "ink")
    c.rect(18, 5, 1, 3, "ink")
    gable(c, 10, 0, 12, 4, SLATE_ROOF)
    c.rect(10, 9, 12, 1, "slate")
    c.points([(5, 5), (4, 6), (26, 5), (27, 6)], "glow")
    return k.finish(c)


def guild_hall(k):
    c = k.new()
    k.mound(c, 30)
    box(c, 3, 13, 26, 13, STONE)
    for x in (6, 11, 20, 25):
        box(c, x, 15, 2, 10, PLASTER)
    box(c, 13, 17, 6, 9, WOOD)
    c.rect(13, 17, 6, 1, "pale stone")
    gable(c, 1, 3, 30, 10, ROOF)
    c.rect(1, 12, 30, 1, "bark")
    k.clump(c, 16, 8, 3, 3, ["ember", "amber", "lantern gold", "glow"])
    flag(c, 16, 0, ("arcane violet", "arcane violet"))
    return k.finish(c)


def bard_stage(k):
    c = k.new()
    k.mound(c, 30)
    box(c, 3, 19, 26, 6, WOOD)
    for x in range(4, 28, 4):
        c.line(x, 20, x, 24, "bark")
    box(c, 3, 6, 2, 13, WOOD)
    box(c, 27, 6, 2, 13, WOOD)
    c.rect(3, 4, 26, 3, "banner red")
    c.rect(3, 4, 26, 1, "berry")
    for x0, d in ((5, 1), (26, -1)):
        for j in range(12):
            w = max(1, 5 - j // 3)
            for i in range(w):
                c.px(x0 + i * d, 7 + j, "arcane violet" if i else "night plum")
    # a lute on its stand and two floating notes
    k.clump(c, 15, 15, 3, 3, ["bark", "earth", "dry path"])
    c.line(16, 13, 20, 8, "bark")
    c.px(15, 15, "ink")
    c.points([(10, 10), (10, 11), (11, 9), (22, 12), (22, 13), (23, 11)], "lantern gold")
    return k.finish(c)


def harbor(k):
    c = k.new()
    k.sh.clump(c, 15.5, 25, 15, 5, WATER, [0.25, 0.5, 0.8])
    box(c, 1, 20, 14, 3, WOOD)
    for x in (2, 7, 12):
        c.line(x, 23, x, 27, "bark")
    # a small boat with a sail
    for j in range(4):
        c.rect(15 + j, 21 + j, 13 - 2 * j, 1, "earth" if j == 0 else "bark")
    c.rect(15, 21, 13, 1, "dry path")
    c.line(21, 6, 21, 20, "bark")
    for j in range(12):
        c.rect(22, 7 + j, 1 + (j * 5) // 11, 1, "parchment white")
        c.rect(20 - (j * 4) // 11, 8 + j, (j * 4) // 11, 1, "sand")
    c.points([(4, 27), (5, 27), (25, 28), (26, 28)], "mist")
    return k.finish(c)


def city_gate(k):
    c = k.new()
    k.mound(c, 30)
    box(c, 2, 8, 9, 19, STONE)
    box(c, 21, 8, 9, 19, STONE)
    box(c, 9, 12, 14, 15, STONE)
    for x in (2, 5, 8, 21, 24, 27):
        box(c, x, 5, 2, 3, STONE)
    for y in range(16, 27):
        for x in range(12, 20):
            if ((x - 15.5) / 4) ** 2 + ((y - 20) / 4) ** 2 < 1 or y >= 20:
                c.px(x, y, "night plum")
    for x in range(12, 20, 2):
        c.line(x, 17, x, 23, "slate")
    c.rect(12, 20, 8, 1, "slate")
    window(c, 5, 13, 2, 3)
    window(c, 25, 13, 2, 3)
    flag(c, 6, 0)
    flag(c, 25, 0)
    return k.finish(c)


def mountain_pass(k):
    c = k.new()
    peaks = ((10, 3, 12), (22, 7, 10))
    for px, top, half in peaks:
        for y in range(top, 28):
            w = ((y - top) * half) // (28 - top) + 1
            for x in range(px - w, px + w):
                lit = x < px
                snow = y < top + 6
                c.px(x, y, ("parchment white" if lit else "mist") if snow else (
                    "pale stone" if lit else "stone"))  # fmt: skip
    for x in range(0, 32):
        for y in range(26, 30):
            if c.opaque(x, y - 3) or y < 28:
                c.px(x, y, "moss" if x < 20 else "moss shadow")
    for y in range(18, 30):
        x = 16 + (y - 18) // 2 + (2 if y > 24 else 0)
        c.rect(x - 1, y, 3, 1, "dry path")
    return k.finish(c)


def great_tree(k):
    c = k.new()
    k.mound(c, 30)
    box(c, 13, 16, 6, 11, WOOD)
    c.line(13, 26, 10, 27, "bark")
    c.line(18, 26, 21, 27, "bark")
    canopy = [(10, 11, 8, 6), (22, 11, 8, 6), (16, 6, 9, 6), (16, 13, 10, 5)]
    for cx, cy, rx, ry in canopy:
        k.clump(c, cx, cy, rx, ry, ["deep forest", "moss shadow", "moss", "leaf", "sunlit leaf"],
                (0.18, 0.42, 0.66, 0.86))  # fmt: skip
    k.finish(c)
    c.stamp(k.lantern.crop(1, 3, 7, 7), 4, 13)
    c.stamp(k.lantern.crop(1, 3, 7, 7), 22, 14)
    return c


def castle(k):
    c = k.new()
    k.mound(c, 30)
    box(c, 7, 13, 18, 14, STONE)
    box(c, 2, 9, 7, 18, STONE)
    box(c, 23, 9, 7, 18, STONE)
    box(c, 12, 6, 8, 10, STONE)
    for x in (2, 6, 23, 27, 7, 11, 21, 12, 17):
        box(c, x, 7 if x in (2, 6, 23, 27) else 11, 3, 2, STONE)
    gable(c, 11, 0, 10, 6, ("berry", "banner red", "dusk violet-grey"))
    for y in range(20, 27):
        for x in range(13, 19):
            if ((x - 15.5) / 3) ** 2 + ((y - 21) / 2) ** 2 < 1 or y >= 21:
                c.px(x, y, "night plum")
    window(c, 4, 13, 2, 3)
    window(c, 26, 13, 2, 3)
    window(c, 15, 9, 2, 3)
    flag(c, 5, 1, ("lantern gold", "amber"))
    flag(c, 26, 1, ("lantern gold", "amber"))
    return k.finish(c)


LANDMARKS = [
    campfire, signpost, crossroads, market_stall,
    inn, bell_tower, training_dummies, watchtower,
    treasure_chest, quest_board, banner_camp, bridge,
    lighthouse, guild_hall, bard_stage, harbor,
    city_gate, mountain_pass, great_tree, castle,
]  # fmt: skip


def draw(ctx):
    k = Kit(ctx)
    return [fn(k) for fn in LANDMARKS]
