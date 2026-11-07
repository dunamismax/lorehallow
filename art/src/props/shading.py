"""Shared shading for props: form-lit clumps and ramps.

A clump is an ellipse. Each pixel gets a ramp color from how much its
surface faces the light, which comes from the top-left and from above.
The ramps are flat palette steps, so this gives 3 or 4 bands per clump,
lit on its top-left and shaded on its bottom-right: form shading, not
pillow shading. Clumps drawn later sit in front.
"""

import math

LIGHT = (-0.55, -0.62, 0.56)  # toward the light: left, up, out of the screen
_n = math.sqrt(sum(v * v for v in LIGHT))
LIGHT = tuple(v / _n for v in LIGHT)

FOLIAGE = ["deep forest", "moss shadow", "moss", "leaf", "sunlit leaf"]
STONE = ["dusk violet-grey", "slate", "stone", "pale stone"]
MOSS = ["moss shadow", "moss", "leaf", "sunlit leaf"]


def lit(nx, ny):
    """Lambert light for a point on a unit dome at (nx, ny)."""
    d2 = nx * nx + ny * ny
    nz = math.sqrt(max(0.0, 1.0 - d2))
    return max(0.0, nx * LIGHT[0] + ny * LIGHT[1] + nz * LIGHT[2])


def band(value, ramp, cuts):
    """Ramp color for a light value; cuts are ascending thresholds."""
    k = sum(value >= c for c in cuts)
    return ramp[min(k, len(ramp) - 1)]


def clump(c, cx, cy, rx, ry, ramp, cuts, mask=None):
    """Draw one form-lit ellipse clump centered on (cx, cy)."""
    for y in range(math.floor(cy - ry), math.ceil(cy + ry) + 1):
        for x in range(math.floor(cx - rx), math.ceil(cx + rx) + 1):
            nx, ny = (x - cx) / rx, (y - cy) / ry
            if nx * nx + ny * ny > 1.0:
                continue
            if mask is not None and not mask(x, y):
                continue
            c.px(x, y, band(lit(nx, ny), ramp, cuts))


def prune(c, passes=2):
    """Remove 1 px nubs (opaque pixels with at most one opaque 4-neighbor),
    which an outline would turn into ink ticks."""
    for _ in range(passes):
        nubs = [
            (x, y)
            for y in range(c.h)
            for x in range(c.w)
            if c.opaque(x, y)
            and sum(c.opaque(x + dx, y + dy) for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1))) <= 1
        ]
        for x, y in nubs:
            c.px(x, y, None)


def bumps(c, rng, colors, count, top_only=True):
    """Small leafy bumps along a silhouette's edge: 2 px nubs that break
    up the smooth ellipse rim."""
    edge = [
        (x, y)
        for y in range(1, c.h - 1)
        for x in range(1, c.w - 1)
        if c.get(x, y) in colors and not c.opaque(x, y - 1)
    ]
    if not top_only:
        edge += [
            (x, y)
            for y in range(1, c.h - 1)
            for x in range(1, c.w - 1)
            if c.get(x, y) in colors and not c.opaque(x - 1, y)
        ]
    for _ in range(count):
        if not edge:
            return
        x, y = rng.choice(edge)
        col = c.get(x, y)
        c.points([(x, y - 1), (x + 1, y - 1)], col)
