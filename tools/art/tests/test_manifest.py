import pytest

from lorehallow_art.manifest import ManifestError, parse_manifest, parse_map


def base(**asset):
    a = {"id": "grass", "target": "tiles/grass.png", "kind": "tile", "size": [32, 32]}
    a.update(asset)
    return {"batch": "01", "assets": [a]}


def test_minimal_tile_and_defaults():
    m = parse_manifest(base())
    a = m.assets[0]
    assert (a.kind, a.size, a.key_color, a.palette) == (
        "tile",
        (32, 32),
        (255, 0, 255),
        "art/style/palette.hex",
    )
    assert a.frame_names() == ["grass"]


def test_sheet_frame_names_from_rows_and_columns():
    m = parse_manifest(
        base(
            id="player_base",
            target="chars/player_base.png",
            kind="sheet",
            size=[64, 96],
            grid={"cell": [32, 48], "columns": 2, "rows": 2},
            frames={
                "count": 4,
                "rows": ["down", "up"],
                "columns": ["idle_0", "idle_1"],
                "baseline": 45,
            },
            pivot=[16, 45],
        )
    )
    assert m.assets[0].frame_names() == ["down_idle_0", "down_idle_1", "up_idle_0", "up_idle_1"]


@pytest.mark.parametrize(
    ("change", "message"),
    [
        ({"id": "Grass"}, "snake_case"),
        ({"target": "grass.png"}, "target"),
        ({"kind": "mesh"}, "kind"),
        ({"size": [32]}, "size"),
        ({"kind": "tileset"}, "needs a grid"),
        ({"grid": {"cell": [32, 32], "columns": 1, "rows": 1}}, "takes no grid"),
        (
            {"kind": "tileset", "grid": {"cell": [32, 32], "columns": 2, "rows": 1}},
            "is not the size",
        ),
        ({"downsample": "bilinear"}, "downsample"),
        ({"key_color": "#12"}, "hex"),
        ({"colour": "red"}, "unknown keys"),
    ],
)
def test_rejects_bad_assets(change, message):
    with pytest.raises(ManifestError, match=message):
        parse_manifest(base(**change))


def test_rejects_duplicates_and_bad_batch():
    data = base()
    data["assets"].append(dict(data["assets"][0]))
    with pytest.raises(ManifestError, match="duplicate"):
        parse_manifest(data)
    with pytest.raises(ManifestError, match="two-digit"):
        parse_manifest({"batch": 1, "assets": base()["assets"]})


def test_map_entries_and_errors():
    m = parse_manifest(base())
    imap = parse_map(
        {"files": {"a.png": "grass", "b.png": {"asset": "grass", "crop": [0, 0, 10, 10]}}}, m
    )
    assert [(e.file, e.asset, e.crop) for e in imap.entries] == [
        ("a.png", "grass", None),
        ("b.png", "grass", (0, 0, 10, 10)),
    ]
    with pytest.raises(ManifestError, match="unknown asset"):
        parse_map({"files": {"a.png": "nope"}}, m)
    with pytest.raises(ManifestError, match="crop"):
        parse_map({"files": {"a.png": {"asset": "grass", "crop": [1, 2]}}}, m)
