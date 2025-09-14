# GroveGrid

[![Live demo](https://img.shields.io/badge/demo-GitHub%20Pages-2ea44f)](https://aplgr.github.io/grovegrid/)
[![Release](https://img.shields.io/github/v/release/aplgr/grovegrid)](https://github.com/aplgr/grovegrid/releases)
![Status](https://img.shields.io/badge/status-alpha-orange)
![Scope](https://img.shields.io/badge/scope-personal%20tool-blue)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[![Go Reference](https://pkg.go.dev/badge/github.com/aplgr/grovegrid.svg)](https://pkg.go.dev/github.com/aplgr/grovegrid)
[![Go Report Card](https://goreportcard.com/badge/github.com/aplgr/grovegrid)](https://goreportcard.com/report/github.com/aplgr/grovegrid)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](#)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/aplgr/grovegrid/issues)


**CSV → interactive heatmaps & timeline for tree rows** (Go CLI · Alpine.js · ECharts)

I keep snapshots of a small, planned grove: rows of trees, each with a position, a numeric condition (0 = dead, higher = better), height in cm, and a short species code. I first rendered static PNGs in Python, but that felt rigid. **GroveGrid** is my tiny CLI tool that turns those CSVs into one self-contained HTML: a calm heatmap with a time slider. It helps me review growth at a glance — and if you track rows of plants or orchards, it might be useful to you as well.

## What it does

* **Color-coded grid** for your rows (row × tree number). Numeric condition is shown as color; missing cells render as *no data* (dark grey), condition **0** as grey.
* **Size-coded points** for height (cm) with helpful tooltips (species, exact values).
* **Switch time slices** (one CSV per slice) via dropdown or slider — no page reloads.
* **Ragged rows** are fine: each row can have a different length.
* **Single, offline HTML** you can open anywhere.

## Quickstart

```bash
# Build
go build -o ./bin/grovegrid ./cmd/grovegrid

# Run (reads all *.csv in the folder)
./bin/grovegrid -in ./data -out ./out -title "GroveGrid"

# Open the result in your browser
# -> ./out/index.html
```

> **Tip:** `data/` can contain multiple files like `2023-04.csv`, `2023-05.csv` etc.
> Every file becomes one time slice. It ships with sample inputs so you can play immediately.

## Data model

**CSV columns (case-insensitive; German header variants are accepted):**

* `row` *(int)* — row index (1..X)
* `position` *(int)* — position in row (1..Y)
* `condition` *(float)* — **0 = dead**, **> 0 = better**; empty/unknown → *no data* (internally `-1`)
* `height` *(float, cm)*
* `species` *(string, short code)* — e.g. `Y`, `Cu`, `P`

**Example**

```csv
row;position;condition;height;species;notes
1;1;3.2;35;Y;new planting 2023-03
1;2;0;28;Y;storm damage (May)
1;3;1.4;15;Cu;
2;1;4.0;42;P;shaded area
```

## Design notes

* **Chart mapping**: ECharts heatmap encodes *condition*; scatter encodes *height* via `symbolSize`.
* **Color logic**: piecewise mapping on the heatmap — `-1` (*no data*), `0` (*dead*, `ZeroColor`), and a red → yellow → green gradient (`GradColors`) for values `> 0`.
* **Stable timeline**: points are keyed by `(row, position)` across months; updates use ECharts’ merge behavior (`setOption(..., false)`), so points don’t jump — only size/color change with a short linear animation.
* **Alpine glue**: the ECharts instance lives outside Alpine’s proxy to avoid recursion and keep reactivity simple.
* **CSV parsing**: delimiter autodetection (`;`, `,`, tab), header normalization (umlauts, dashes/underscores), robust float parsing (`,` and `.`), and optional mapping from legacy text labels to numeric condition.
* **Ragged rows handling**: the full grid is rendered; missing coordinates are filled as *no data*.

## CLI Flags

| Flag     | Default     | Description                                            |
| -------- | ----------- | ------------------------------------------------------ |
| `-in`    | `./data`    | Input directory with CSV files (each file = one slice) |
| `-out`   | `./out`     | Output directory (will be created)                     |
| `-title` | `GroveGrid` | Page title for the generated HTML                      |
| `-json-out` | *(empty)* | If set, also writes the raw data as JSON to this path |


## Ideas

* Optional color schemes
* Export PNG/SVG from the HTML
* Small `-serve` flag to share over LAN
* Tunable condition binning strategies via CLI flags

## License

MIT — see `LICENSE`.
