extends GutTest


func test_build_label_uses_version_and_short_commit() -> void:
	var original_version: String = Config.version
	var original_commit: String = Config.commit
	Config.version = "v0.2.0"
	Config.commit = "a1b2c3d4e5f6"
	assert_eq(Config.build_label(), "v0.2.0 (a1b2c3d)")
	Config.version = original_version
	Config.commit = original_commit


func test_short_commit_handles_short_strings() -> void:
	var original_commit: String = Config.commit
	Config.commit = "abc"
	assert_eq(Config.short_commit(), "abc")
	Config.commit = original_commit


func test_log_format_sorts_fields() -> void:
	var line: String = Log.format_line(Log.Level.INFO, "net", "connected", {"b": 2, "a": 1})
	assert_eq(line, "INFO [net] connected a=1 b=2")


func test_title_screen_shows_build_label() -> void:
	var scene: PackedScene = load("res://ui/title_screen.tscn")
	var title: Control = scene.instantiate()
	add_child_autofree(title)
	var label: Label = title.get_node("%VersionLabel")
	assert_eq(label.text, Config.build_label())
