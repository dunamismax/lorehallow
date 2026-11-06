"""Command-line entry point: `lorehallow-art intake|sync`."""

from __future__ import annotations

import argparse
import sys


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="lorehallow-art")
    parser.add_subparsers(dest="command", required=True)
    parser.parse_args(argv)
    return 0


if __name__ == "__main__":
    sys.exit(main())
