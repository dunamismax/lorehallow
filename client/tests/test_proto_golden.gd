extends GutTest
## Golden round-trip tests: every vector Go wrote to proto/testdata/ must
## decode to the values in its .json description and re-encode to the
## exact same bytes. See server/internal/protovectors.

const EnvelopePb: GDScript = preload("res://net/pb/envelope_pb.gd")
const PbDict: GDScript = preload("res://tests/helpers/pb_dict.gd")


func _testdata_dir() -> String:
	return ProjectSettings.globalize_path("res://").path_join("../proto/testdata").simplify_path()


func _vector_names() -> PackedStringArray:
	var names: PackedStringArray = []
	for file: String in DirAccess.get_files_at(_testdata_dir()):
		if file.ends_with(".bin"):
			names.append(file.get_basename())
	names.sort()
	return names


func test_vectors_exist() -> void:
	assert_gt(_vector_names().size(), 0, "no golden vectors found; run make proto")


func test_golden_round_trip() -> void:
	for name: String in _vector_names():
		var dir: String = _testdata_dir()
		var raw: PackedByteArray = FileAccess.get_file_as_bytes(dir.path_join(name + ".bin"))
		var json_text: String = FileAccess.get_file_as_string(dir.path_join(name + ".json"))
		var expected: Variant = JSON.parse_string(json_text)
		assert_true(expected is Dictionary, "%s.json is not an object" % name)

		var env: Object = EnvelopePb.Envelope.new()
		var rc: int = env.call("from_bytes", raw)
		assert_eq(rc, 0, "%s: decode error %d" % [name, rc])

		var got: Dictionary = PbDict.from_message(env)
		var want: Dictionary = PbDict.from_json(expected)
		assert_eq_deep(got, want)

		var again: PackedByteArray = env.call("to_bytes")
		assert_eq(again.hex_encode(), raw.hex_encode(), "%s: re-encoded bytes differ" % name)


func test_protocol_version_is_positive() -> void:
	assert_gt(ProtocolVersion.CURRENT, 0)
