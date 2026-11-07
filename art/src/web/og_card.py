"""Open Graph card background, 240×126: the generated painting of hills at
dusk (og_card_painted) with its old generated traveler painted out and the
hand-drawn hero on the path, carrying a lantern toward the village.
"""

BOX = (133, 74, 158, 107)  # the old traveler
X, FEET = 145, 105


def draw(ctx):
    scene = ctx.module("scene.py")
    c = ctx.art("og_card_painted")
    scene.paint_out(c, *BOX)
    hero = ctx.art("concept_hero")
    hero.stamp(ctx.pxl("../props/lantern.pxl.yaml")[0], 21, 32)
    scene.stand(c, hero, X, FEET)
    return c
