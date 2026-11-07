extends SceneTree
## Renders the dev art scene for real and checks it is pixel-perfect: every
## sprite on screen must equal its source image scaled up by nearest
## neighbor, pixel for pixel, at 1x to 4x, including the 3×3 seam patches.
## Blur, filtering, sub-pixel offsets, seams or color shifts all show up as
## mismatches. Needs a real renderer (not --headless):
##
##   make client-art-check            # tiles, chars and props
##   make client-art-check ART=all    # every synced image (slower)
##
## Exits 0 when every checked pixel matches, 1 otherwise.

const SCENE: String = "res://scenes/dev/art_test.tscn"
const DEFAULT_CATEGORIES: PackedStringArray = ["tiles", "chars", "props"]

var _checked: int = 0
var _bad: int = 0


func _initialize() -> void:
	_run.call_deferred()


func _run() -> void:
	var categories: PackedStringArray = DEFAULT_CATEGORIES
	for arg: String in OS.get_cmdline_user_args():
		if arg.begins_with("--art="):
			var value: String = arg.trim_prefix("--art=")
			categories = PackedStringArray() if value == "all" else value.split(",")
	var scene: Node2D = (load(SCENE) as PackedScene).instantiate()
	root.add_child(scene)
	await process_frame
	var camera: Camera2D = null
	for child: Node in scene.get_children():
		if child is Camera2D:
			camera = child
	var view: Vector2i = root.get_visible_rect().size
	var sprites: int = 0
	for child: Node in scene.get_children():
		if not child is Sprite2D:
			continue
		var sprite: Sprite2D = child
		var rel: String = sprite.texture.resource_path.trim_prefix("res://assets/art/")
		if not categories.is_empty() and not categories.has(rel.get_slice("/", 0)):
			continue
		sprites += 1
		await _check_sprite(sprite, camera, view)
	print(
		(
			"art_verify: %d sprites, %d pixels checked, %d mismatched, viewport %dx%d"
			% [sprites, _checked, _bad, view.x, view.y]
		)
	)
	quit(1 if _bad > 0 or _checked == 0 else 0)


func _check_sprite(sprite: Sprite2D, camera: Camera2D, view: Vector2i) -> void:
	var source: Image = sprite.texture.get_image()
	if source.is_compressed():
		source.decompress()
	var region: Rect2i = Rect2i(Vector2i.ZERO, source.get_size())
	if sprite.region_enabled:
		region = Rect2i(sprite.region_rect)
	var s: int = int(sprite.scale.x)
	var size: Vector2i = region.size * s
	var origin: Vector2i = Vector2i(sprite.position)
	var mismatched: int = 0
	for oy: int in range(0, size.y, view.y):
		for ox: int in range(0, size.x, view.x):
			camera.position = Vector2(origin + Vector2i(ox, oy))
			await RenderingServer.frame_post_draw
			await RenderingServer.frame_post_draw
			var shot: Image = root.get_texture().get_image()
			for y: int in mini(view.y, size.y - oy):
				for x: int in mini(view.x, size.x - ox):
					var src: Color = source.get_pixel(
						region.position.x + (ox + x) / s, region.position.y + (oy + y) / s
					)
					if src.a8 == 0:
						continue
					_checked += 1
					if shot.get_pixel(x, y).to_rgba32() != src.to_rgba32():
						mismatched += 1
	if mismatched > 0:
		_bad += mismatched
		print(
			(
				"art_verify: %s at %dx: %d pixels differ"
				% [sprite.texture.resource_path, s, mismatched]
			)
		)
