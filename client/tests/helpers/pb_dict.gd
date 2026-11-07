extends RefCounted
## Converts godobuf messages and protojson descriptions into one comparable
## shape, so golden tests can check that decoding produced the right values.
##
## Rules, matching protojson with proto field names:
## - fields at their proto3 default (0, "", false, empty list) are omitted
## - a set sub-message is a nested Dictionary, even when empty
## - every scalar becomes a String (protojson writes 64-bit ints as strings)
## - enums are numbers (the vectors use protojson's UseEnumNumbers)
## - repeated messages are Arrays of Dictionaries


## Returns a normalized Dictionary for a godobuf message object.
static func from_message(msg: Object) -> Dictionary:
	var out: Dictionary = {}
	var data: Dictionary = msg.get("data")
	for tag: int in data:
		var service: Object = data[tag]
		var field: Object = service.get("field")
		var value: Variant = field.get("value")
		var name: String = field.get("name")
		if value == null:
			continue
		if value is Object:
			out[name] = from_message(value)
		elif value is Array:
			if not (value as Array).is_empty():
				out[name] = (value as Array).map(_element)
		elif not _is_default(value):
			out[name] = _scalar(value)
	return out


## Returns a normalized Dictionary for parsed protojson.
static func from_json(value: Dictionary) -> Dictionary:
	var out: Dictionary = {}
	for key: String in value:
		var v: Variant = value[key]
		if v is Dictionary:
			out[key] = from_json(v)
		elif v is Array:
			out[key] = (v as Array).map(_json_element)
		else:
			out[key] = _json_scalar(v)
	return out


static func _is_default(value: Variant) -> bool:
	match typeof(value):
		TYPE_INT:
			return value == 0
		TYPE_FLOAT:
			return value == 0.0
		TYPE_BOOL:
			return value == false
		TYPE_STRING:
			return value == ""
		TYPE_PACKED_BYTE_ARRAY:
			return (value as PackedByteArray).is_empty()
	return false


static func _element(value: Variant) -> Variant:
	if value is Object:
		return from_message(value)
	return _scalar(value)


static func _json_element(value: Variant) -> Variant:
	if value is Dictionary:
		return from_json(value)
	return _json_scalar(value)


static func _scalar(value: Variant) -> String:
	if value is PackedByteArray:
		return Marshalls.raw_to_base64(value)
	return str(value)


static func _json_scalar(value: Variant) -> String:
	# Godot parses every JSON number as float; whole numbers are ints here.
	if value is float and is_equal_approx(fmod(value, 1.0), 0.0) and absf(value) < 9.0e15:
		return str(int(value))
	return str(value)
