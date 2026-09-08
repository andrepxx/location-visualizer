#!/usr/bin/env python3

import argparse

from pathlib import Path

def main():

	parser = argparse.ArgumentParser(
		description="Generate an SVG social preview from a template."
	)

	parser.add_argument("input", type=Path, help="Input SVG template")
	parser.add_argument("output", type=Path, help="Output SVG file")
	parser.add_argument("stars", help="Number of GitHub stars")
	parser.add_argument("forks", help="Number of GitHub forks")
	parser.add_argument("version", help="Project version")

	args = parser.parse_args()

	# Read the SVG template.
	svg = args.input.read_text(encoding="utf-8")

	# Replace template placeholders.
	replacements = {
		"{{stars}}": args.stars,
		"{{forks}}": args.forks,
		"{{version}}": args.version,
	}

	for placeholder, value in replacements.items():
		svg = svg.replace(placeholder, value)

	# Write the generated SVG.
	args.output.write_text(svg, encoding="utf-8")


if __name__ == "__main__":
    main()
