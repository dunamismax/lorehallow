extends Node
## Small structured logger. Every line is "LEVEL [area] message key=value".
## Later phases add a debug console sink; for now everything goes to stdout.

enum Level { DEBUG, INFO, WARN, ERROR }

var level: Level = Level.INFO


func debug(area: String, message: String, fields: Dictionary = {}) -> void:
	_write(Level.DEBUG, area, message, fields)


func info(area: String, message: String, fields: Dictionary = {}) -> void:
	_write(Level.INFO, area, message, fields)


func warn(area: String, message: String, fields: Dictionary = {}) -> void:
	_write(Level.WARN, area, message, fields)


func error(area: String, message: String, fields: Dictionary = {}) -> void:
	_write(Level.ERROR, area, message, fields)


## Formats one log line. Public so tests can check the format.
func format_line(lvl: Level, area: String, message: String, fields: Dictionary) -> String:
	var parts: PackedStringArray = [Level.keys()[lvl], "[%s]" % area, message]
	var keys: Array = fields.keys()
	keys.sort()
	for key: Variant in keys:
		parts.append("%s=%s" % [str(key), str(fields[key])])
	return " ".join(parts)


func _write(lvl: Level, area: String, message: String, fields: Dictionary) -> void:
	if lvl < level:
		return
	var line: String = format_line(lvl, area, message, fields)
	if lvl >= Level.WARN:
		printerr(line)
	else:
		print(line)
