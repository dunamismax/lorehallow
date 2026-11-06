extends Node
## Client configuration and build information.
##
## The build version and commit come from project settings, which
## `make client` and exports override through override.cfg so the
## binary always knows which build it is.

const DEFAULT_VERSION: String = "dev"
const DEFAULT_COMMIT: String = "unknown"

var version: String = DEFAULT_VERSION
var commit: String = DEFAULT_COMMIT


func _ready() -> void:
	version = str(ProjectSettings.get_setting("application/config/version", DEFAULT_VERSION))
	commit = str(ProjectSettings.get_setting("lorehallow/build/commit", DEFAULT_COMMIT))
	Log.info("config", "client starting", {"version": version, "commit": short_commit()})


## The first 7 characters of the commit, or the whole string if shorter.
func short_commit() -> String:
	return commit.substr(0, 7)


## Human-readable build string for the title screen, e.g. "v0.2.0 (a1b2c3d)".
func build_label() -> String:
	return "%s (%s)" % [version, short_commit()]
