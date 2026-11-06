extends GutTest
## Synced art must import as pixel art and stay on the locked palette.

const ArtTest: GDScript = preload("res://scenes/dev/art_test.gd")


func _palette() -> Dictionary:
	var path: String = ProjectSettings.globalize_path("res://").path_join(
		"../art/style/palette.hex"
	)
	var colors: Dictionary = {}
	for raw: String in FileAccess.get_file_as_string(path).split("\n"):
		var line: String = raw.split(";")[0].strip_edges()
		if line.length() == 6:
			colors[line.to_lower()] = true
	return colors


func test_art_is_present() -> void:
	assert_gt(ArtTest.find_images("res://assets/art").size(), 0, "run make sync")


func test_imports_use_pixel_art_settings() -> void:
	for path: String in ArtTest.find_images("res://assets/art"):
		var cfg: ConfigFile = ConfigFile.new()
		assert_eq(cfg.load(path + ".import"), OK, "%s has no .import file" % path)
		assert_eq(cfg.get_value("params", "compress/mode", -1), 0, "%s: not lossless" % path)
		assert_false(cfg.get_value("params", "mipmaps/generate", true), "%s: mipmaps on" % path)


func test_pixels_are_on_palette() -> void:
	var palette: Dictionary = _palette()
	assert_gt(palette.size(), 0)
	for path: String in ArtTest.find_images("res://assets/art"):
		var texture: Texture2D = load(path)
		var image: Image = texture.get_image()
		var off: int = 0
		for y: int in image.get_height():
			for x: int in image.get_width():
				var c: Color = image.get_pixel(x, y)
				if c.a8 == 0:
					continue
				if c.a8 != 255 or not palette.has(c.to_html(false)):
					off += 1
		assert_eq(off, 0, "%s has %d off-palette or semi-transparent pixels" % [path, off])


func test_art_test_scene_draws_at_integer_scales() -> void:
	var scene: PackedScene = load("res://scenes/dev/art_test.tscn")
	var root: Node2D = scene.instantiate()
	add_child_autofree(root)
	var sprites: int = 0
	for child: Node in root.get_children():
		if child is Sprite2D:
			sprites += 1
			var s: Vector2 = (child as Sprite2D).scale
			assert_eq(s.x, roundf(s.x))
			assert_eq(s.x, s.y)
			assert_eq((child as Sprite2D).position, (child as Sprite2D).position.round())
	assert_gt(sprites, 0)
