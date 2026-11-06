extends Node2D
## Dev scene: every image in res://assets/art drawn at 1x, 2x, 3x and 4x,
## plus a 3×3 patch of each tile from every tileset to show seams.
## Run it from the editor (F6 on this scene). Arrow keys or the mouse
## wheel pan the camera; the camera stays on whole pixels.

const ART_ROOT: String = "res://assets/art"
const SCALES: Array[int] = [1, 2, 3, 4]
const GAP: int = 8
const LABEL_HEIGHT: int = 10
const PAN_STEP: int = 32

var _camera: Camera2D
var _cursor: Vector2i = Vector2i(GAP, GAP)
var _row_height: int = 0


func _ready() -> void:
	for path: String in find_images(ART_ROOT):
		_add_image(path)
	_camera = Camera2D.new()
	_camera.anchor_mode = Camera2D.ANCHOR_MODE_FIXED_TOP_LEFT
	add_child(_camera)


func _unhandled_input(event: InputEvent) -> void:
	var step: Vector2i = Vector2i.ZERO
	if event.is_action_pressed("ui_down"):
		step.y = 1
	elif event.is_action_pressed("ui_up"):
		step.y = -1
	elif event.is_action_pressed("ui_right"):
		step.x = 1
	elif event.is_action_pressed("ui_left"):
		step.x = -1
	elif event is InputEventMouseButton and (event as InputEventMouseButton).pressed:
		var button: MouseButton = (event as InputEventMouseButton).button_index
		if button == MOUSE_BUTTON_WHEEL_DOWN:
			step.y = 1
		elif button == MOUSE_BUTTON_WHEEL_UP:
			step.y = -1
	if step != Vector2i.ZERO:
		var pos: Vector2i = Vector2i(_camera.position) + step * PAN_STEP
		_camera.position = Vector2(maxi(pos.x, 0), maxi(pos.y, 0))


## Every .png under root, sorted, recursing into folders.
static func find_images(root: String) -> PackedStringArray:
	var out: PackedStringArray = []
	var dir: DirAccess = DirAccess.open(root)
	if dir == null:
		return out
	for sub: String in dir.get_directories():
		out.append_array(find_images(root.path_join(sub)))
	for file: String in dir.get_files():
		if file.get_extension() == "png":
			out.append(root.path_join(file))
	out.sort()
	return out


func _add_image(path: String) -> void:
	var texture: Texture2D = load(path)
	if texture == null:
		Log.warn("art_test", "could not load", {"path": path})
		return
	_new_row()
	var label: Label = Label.new()
	label.text = path.trim_prefix(ART_ROOT + "/")
	label.position = Vector2(_cursor)
	label.add_theme_font_size_override("font_size", 8)
	add_child(label)
	_cursor.y += LABEL_HEIGHT
	for s: int in SCALES:
		var sprite: Sprite2D = Sprite2D.new()
		sprite.texture = texture
		sprite.centered = false
		sprite.scale = Vector2(s, s)
		sprite.position = Vector2(_cursor)
		add_child(sprite)
		_cursor.x += texture.get_width() * s + GAP
		_row_height = maxi(_row_height, texture.get_height() * s)
	var sidecar: Dictionary = load_sidecar(path)
	if sidecar.get("kind", "") == "tileset":
		_add_seam_patch(texture, sidecar)


## The JSON sidecar next to a tileset or sheet, or {} if there is none.
static func load_sidecar(path: String) -> Dictionary:
	var json_path: String = path.get_basename() + ".json"
	if not FileAccess.file_exists(json_path):
		return {}
	var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(json_path))
	return parsed if parsed is Dictionary else {}


func _add_seam_patch(texture: Texture2D, sidecar: Dictionary) -> void:
	# The first tile repeated 3×3 at 2x: seams show up as visible lines.
	var size: Array = sidecar.get("frame_size", [32, 32])
	var tile: Vector2i = Vector2i(int(size[0]), int(size[1]))
	for y: int in 3:
		for x: int in 3:
			var sprite: Sprite2D = Sprite2D.new()
			sprite.texture = texture
			sprite.centered = false
			sprite.region_enabled = true
			sprite.region_rect = Rect2(Vector2.ZERO, Vector2(tile))
			sprite.scale = Vector2(2, 2)
			sprite.position = Vector2(_cursor + Vector2i(x, y) * tile * 2)
			add_child(sprite)
	_cursor.x += tile.x * 6 + GAP
	_row_height = maxi(_row_height, tile.y * 6)


func _new_row() -> void:
	if _row_height > 0:
		_cursor = Vector2i(GAP, _cursor.y + _row_height + GAP)
	_row_height = 0
