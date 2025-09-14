# data/ - Sample Input for GroveGrid

This folder contains **sample CSV files** so you can try the tool immediately.

- Use them as-is: `go run ./cmd/grovegrid -in ./data -out ./out`  
- Or remove them and place **your own CSVs** here.
- `data/` is **read-only** for the tool. Generated artifacts go to `out/`.

## CSV format

- **Header row** is required. Header names become labels in the UI.
- **Delimiter**: `,` or `;` or `TAB` (auto-detected).
- **Columns** (case-insensitive):
  1. **X** (int, 1-based)
  2. **Y** (int, 1-based)
  3. **Value** (float) - `-1` = no data, `0` = zero, `>0` = value
  4. **Size** (float, optional) - controls point size
  5. **Extras** (any) - keys = header names, shown in tooltip

**Example:**
```csv
X;Y;Value;Size;Comment
1;1;0;10;newly planted
1;2;3.5;12;ok
2;1;-1;0;no measurement
2;2;7.2;15;very good
```

## Slices (filenames)

Each file is one **slice** in the UI.  
The slice label is the **filename without extension** (e.g., `2025-08.csv` → “2025-08”). Chronological naming helps, but it's not required.

