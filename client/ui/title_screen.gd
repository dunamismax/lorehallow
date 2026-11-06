extends Control
## Placeholder title screen: the game name and the build version.

@onready var _version_label: Label = %VersionLabel


func _ready() -> void:
	_version_label.text = Config.build_label()
