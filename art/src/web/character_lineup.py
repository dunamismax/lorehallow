"""Five adventurers at the in-game 32×48 scale on a strip of grass, for the
landing page and /world (192×128). From left to right: a knight, a
ranger, a scholar-mage, a bard and a healer.

Every adventurer is built from the hand-drawn hero: the same head and
body, with region-limited palette swaps for their gear, hair and skin,
plus a hand-drawn gear part (chars/gear.pxl.yaml). That keeps all five
on-model with the game's characters.
"""

HEAD_BOTTOM = 28  # hero rows above this are the head

SKIN = {
    "light": {},
    "medium": {"skin light": "skin mid", "skin mid": "skin deep", "skin deep": "bark"},
    "deep": {"skin light": "skin deep", "skin mid": "bark", "skin deep": "dark soil"},
}

ADVENTURERS = [
    {  # knight: slate armor, banner-red tabard, round shield, plumed helm
        "skin": "medium",
        "hair": {"dry path": "pale stone", "earth": "stone", "bark": "slate"},
        "body": {"sunlit leaf": "pale stone", "leaf": "pale stone", "moss": "stone",
                 "moss shadow": "slate", "parchment white": "berry", "sand": "banner red",
                 "pale stone": "night plum", "dusk blue": "stone", "deep dusk": "slate"},
        "helm": True,
        "gear": [(0, (0, 0, 10, 10), 4, 31, None), (6, (0, 0, 7, 5), 13, 12, None)],
    },
    {  # ranger: moss-green hood and cloak, earthy tunic, a longbow
        "skin": "medium",
        "hair": {"dry path": "leaf", "earth": "moss", "bark": "moss shadow"},
        "body": {"sunlit leaf": "leaf", "leaf": "moss", "moss": "moss shadow",
                 "moss shadow": "deep forest", "parchment white": "sand", "sand": "dry path",
                 "pale stone": "earth", "dusk blue": "earth", "deep dusk": "bark"},
        "gear": [(1, (0, 0, 7, 22), 22, 21, None)],
    },
    {  # scholar-mage: arcane violet robes, white hair, pointed hat, glowing staff
        "skin": "light",
        "hair": {"dry path": "parchment white", "earth": "pale stone", "bark": "stone"},
        "body": {"sunlit leaf": "arcane violet", "leaf": "arcane violet", "moss": "dusk violet-grey",
                 "moss shadow": "night plum", "parchment white": "mist", "sand": "sky",
                 "pale stone": "river", "dusk blue": "arcane violet", "deep dusk": "night plum"},
        "gear": [(3, (0, 0, 16, 12), 8, 10, None), (2, (0, 0, 8, 25), 21, 13, None)],
    },
    {  # bard: amber cloak, red cap with a feather, a lute
        "skin": "deep",
        "hair": {"dry path": "amber", "earth": "ember", "bark": "bark"},
        "body": {"sunlit leaf": "lantern gold", "leaf": "amber", "moss": "ember",
                 "moss shadow": "bark", "dusk blue": "banner red", "deep dusk": "night plum"},
        "gear": [(4, (0, 0, 14, 11), 9, 11, None), (5, (0, 0, 11, 12), 9, 30, None)],
    },
    {  # healer: mist and parchment white, dark hair, a teal glass lantern
        "skin": "deep",
        "hair": {"dry path": "bark", "earth": "dark soil", "bark": "night plum"},
        "body": {"sunlit leaf": "parchment white", "leaf": "mist", "moss": "sky",
                 "moss shadow": "river", "dusk blue": "mist", "deep dusk": "sky"},
        "gear": [("lantern", None, 21, 32, None)],
    },
]  # fmt: skip

TEAL = {"lantern gold": "teal glass", "amber": "teal glass", "glow": "mist", "ember": "river"}
SPACING = 36
FEET = 115  # row the adventurers stand on


def adventurer(ctx, hero, gear, lantern, spec):
    head = hero.crop(0, 0, 32, HEAD_BOTTOM).swap({**spec["hair"], **SKIN[spec["skin"]]})
    body = hero.crop(0, HEAD_BOTTOM, 32, 48 - HEAD_BOTTOM)
    body = body.swap({**spec["body"], **SKIN[spec["skin"]]})
    c = ctx.canvas(32, 48)
    c.stamp(head, 0, 0)
    c.stamp(body, 0, HEAD_BOTTOM)
    if spec.get("helm"):
        # a brow band of steel over the forehead
        for x in range(2, 30):
            if c.get(x, 22) in ("skin mid", "skin deep", "bark", "skin light"):
                c.px(x, 22, "stone")
    for frame, box, x, y, _ in spec["gear"]:
        if frame == "lantern":
            c.stamp(lantern.swap(TEAL), x, y)
        else:
            c.stamp(gear[frame].crop(*box), x, y)
    return c


def strip(ctx):
    """A strip of grass along the bottom, with a leafy top edge and an
    ink outline against the transparent background."""
    s = ctx.canvas(192, 128)
    grass = ctx.frame("terrain_base", "grass_a")
    tufted = ctx.frame("terrain_base", "grass_flowers")
    rng = ctx.rng("strip")
    for x in range(192):
        top = 108 + (1 if (x // 7) % 3 == 0 else 0) + (1 if (x // 3) % 5 == 2 else 0)
        for y in range(top, 128):
            src = tufted if (x // 32) % 3 == 1 else grass
            s.px(x, y, src.get(x % 32, y % 32))
        s.px(x, top, "leaf" if rng.random() < 0.7 else "sunlit leaf")
    s.outline("ink")
    return s


def draw(ctx):
    hero = ctx.art("concept_hero")
    gear = ctx.pxl("../chars/gear.pxl.yaml")
    lantern = ctx.pxl("../props/lantern.pxl.yaml")[0]
    c = strip(ctx)
    x0 = (192 - SPACING * 4 - 32) // 2
    for i, spec in enumerate(ADVENTURERS):
        c.stamp(adventurer(ctx, hero, gear, lantern, spec), x0 + i * SPACING, FEET - 45)
    return c
