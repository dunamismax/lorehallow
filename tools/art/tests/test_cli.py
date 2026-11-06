import pytest

from lorehallow_art.cli import main


def test_requires_command():
    with pytest.raises(SystemExit):
        main([])
