"""A wooden lantern post, 32×64: a post on a stone footing with an arm
and brace, and a lit lantern hanging on a short chain. Stands on its
footing (column 8, row 61)."""


def draw(ctx):
    sh = ctx.module("shading.py")
    c = ctx.canvas()
    # post, lit on the left
    for x, col in zip(range(6, 10), ("dry path", "earth", "bark", "dark soil"), strict=True):
        c.line(x, 6, x, 55, col)
    c.line(7, 20, 7, 30, "bark")
    c.line(8, 38, 8, 46, "dark soil")
    # arm with its top lit
    c.rect(6, 9, 21, 1, "dry path")
    c.rect(6, 10, 21, 1, "earth")
    c.rect(22, 10, 5, 1, "bark")
    # brace
    for k in range(6):
        c.px(10 + k, 17 - k, "earth")
        c.px(10 + k, 18 - k, "bark")
    # stone footing
    sh.clump(c, 8, 57, 6, 4, sh.STONE, [0.2, 0.45, 0.75])
    c.rect(3, 57, 11, 4, "stone")
    c.rect(3, 57, 2, 3, "pale stone")
    c.rect(4, 60, 10, 1, "slate")
    c.rect(11, 57, 3, 3, "slate")
    sh.prune(c)
    c.outline("ink")
    # chain and lantern hang outside the outline
    for y in range(12, 16):
        c.px(21, y, "slate" if y % 2 else "dusk violet-grey")
    c.stamp(ctx.pxl("lantern.pxl.yaml")[0], 17, 16)
    return c
