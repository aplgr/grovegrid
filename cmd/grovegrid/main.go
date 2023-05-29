package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// templatesRoot points to the ./templates folder next to the executable or repo root.
var templatesRoot string

func init() {
	// Try to locate ./templates relative to the executable for "go run" and built binaries
	exe, err := os.Executable()
	if err == nil {
		d := filepath.Dir(exe)
		try := filepath.Join(d, "templates")
		if st, err2 := os.Stat(try); err2 == nil && st.IsDir() {
			templatesRoot = try
		}
	}
	// Fallback to current working directory ./templates
	if templatesRoot == "" {
		cwd, _ := os.Getwd()
		try := filepath.Join(cwd, "templates")
		if st, err2 := os.Stat(try); err2 == nil && st.IsDir() {
			templatesRoot = try
		}
	}
	if templatesRoot == "" {
		templatesRoot = "templates"
	}
}

type Record struct {
	X      int               `json:"x"`
	Y      int               `json:"y"`
	Value  float64           `json:"value"` // -1 means "no data"
	Size   float64           `json:"size"`  // circle size
	Extras map[string]string `json:"extras,omitempty"`
}

type MonthData struct {
	Heat   [][3]float64             `json:"heat"`
	Points []map[string]interface{} `json:"points"`
}

// Labels derived from CSV headers (not hard-coded).
type Labels struct {
	X      string   `json:"x"`
	Y      string   `json:"y"`
	Value  string   `json:"value"`
	Size   string   `json:"size"`
	Extras []string `json:"extras"`
}

type Meta struct {
	XMax        int               `json:"x_max"`
	YMax        int               `json:"y_max"`
	ValueMinPos float64           `json:"value_min_pos"`
	ValueMax    float64           `json:"value_max"`
	ZeroColor   string            `json:"zero_color"`
	NoDataColor string            `json:"nodata_color"`
	GradColors  []string          `json:"grad_colors"`
	SizeMin     float64           `json:"size_min"`
	SizeMax     float64           `json:"size_max"`
	Months      []string          `json:"months"`
	GeneratedAt string            `json:"generated_at"`
	Notes       map[string]string `json:"notes,omitempty"`
	Title       string            `json:"title"`
	Labels      Labels            `json:"labels"`
}

type Output struct {
	Meta     Meta                  `json:"meta"`
	Datasets map[string]*MonthData `json:"datasets"`
}

func main() {
	inDir := flag.String("in", "./data", "Input directory with CSV files (e.g. 2025-01.csv, 2025-02.csv)")
	outDir := flag.String("out", "./out", "Output directory")
	title := flag.String("title", "GroveGrid", "Page title")
	jsonOut := flag.String("json-out", "", "optional path to write JSON data (disabled if empty)")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}

	files, err := filepath.Glob(filepath.Join(*inDir, "*.csv"))
	if err != nil {
		panic(err)
	}
	if len(files) == 0 {
		fmt.Println("No CSV files found in", *inDir)
		return
	}
	sort.Strings(files)

	all := make(map[string][]Record)
	masterHeader := []string{}
	xMax, yMax := 0, 0
	gMin, gMax := 1e12, -1.0
	zMinPos, zMax := 1e12, -1.0

	for _, f := range files {
		month := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		recs, hdr, err := parseCSV(f)
		if err != nil {
			panic(fmt.Errorf("parse %s: %w", f, err))
		}
		if len(masterHeader) == 0 {
			masterHeader = hdr
		}
		all[month] = recs
		for _, r := range recs {
			if r.X > xMax {
				xMax = r.X
			}
			if r.Y > yMax {
				yMax = r.Y
			}
			if r.Size > 0 {
				if r.Size < gMin {
					gMin = r.Size
				}
				if r.Size > gMax {
					gMax = r.Size
				}
			}
			if r.Value > 0 {
				if r.Value < zMinPos {
					zMinPos = r.Value
				}
				if r.Value > zMax {
					zMax = r.Value
				}
			}
		}
	}

	// Fallbacks
	if gMin == 1e12 {
		gMin = 0
		gMax = 0
	}
	if zMinPos == 1e12 {
		zMinPos = 0
	}
	if zMax < 0 {
		zMax = 0
	}

	// Build dynamic labels from CSV header (positions 0..3) and extras
	labels := Labels{X: "X", Y: "Y", Value: "Value", Size: "Size", Extras: []string{}}
	if len(masterHeader) >= 1 {
		labels.X = strings.TrimSpace(masterHeader[0])
	}
	if len(masterHeader) >= 2 {
		labels.Y = strings.TrimSpace(masterHeader[1])
	}
	if len(masterHeader) >= 3 {
		labels.Value = strings.TrimSpace(masterHeader[2])
	}
	if len(masterHeader) >= 4 {
		labels.Size = strings.TrimSpace(masterHeader[3])
	}
	if len(masterHeader) >= 5 {
		for _, h := range masterHeader[4:] {
			labels.Extras = append(labels.Extras, strings.TrimSpace(h))
		}
	}

	out := Output{
		Meta: Meta{
			XMax:        xMax,
			YMax:        yMax,
			ValueMinPos: zMinPos,
			ValueMax:    zMax,
			ZeroColor:   "#555555",
			NoDataColor: "#222222",
			GradColors:  []string{"#d73027", "#fdae61", "#fee08b", "#a6d96a", "#1a9850"},
			SizeMin:     gMin,
			SizeMax:     gMax,
			GeneratedAt: time.Now().Format(time.RFC3339),
			Notes: map[string]string{
				"x_axis":     labels.X + " (1..X)",
				"y_axis":     labels.Y + " (1..Y)",
				"value_info": labels.Value + ": 0=zero, >0 better; <0 no data",
				"size_info":  labels.Size + ": circle size",
			},
			Title:  *title,
			Labels: labels,
		},
		Datasets: map[string]*MonthData{},
	}

	months := make([]string, 0, len(all))
	for m := range all {
		months = append(months, m)
	}
	sort.Strings(months)
	out.Meta.Months = months

	// Build datasets
	for _, m := range months {
		recs := all[m]
		md := &MonthData{}
		present := map[[2]int]Record{}
		for _, r := range recs {
			present[[2]int{r.X, r.Y}] = r
		}

		// full grid: value -1 for "no data" (absent)
		for x := 1; x <= xMax; x++ {
			for y := 1; y <= yMax; y++ {
				val := -1.0
				if r, ok := present[[2]int{x, y}]; ok {
					val = r.Value // 0=zero, >0 better
				}
				md.Heat = append(md.Heat, [3]float64{float64(x), float64(y), val})
			}
		}

		// points: present only
		for _, r := range recs {
			md.Points = append(md.Points, map[string]interface{}{
				"x":      r.X,
				"y":      r.Y,
				"value":  r.Value,
				"size":   r.Size,
				"extras": r.Extras,
			})
		}
		out.Datasets[m] = md
	}

	// optional: write data.json if -json-out is set
	if *jsonOut != "" {
		if err := os.MkdirAll(filepath.Dir(*jsonOut), 0o755); err != nil {
			panic(err)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		if err := os.WriteFile(*jsonOut, b, 0o644); err != nil {
			panic(err)
		}
	}

	// write index.html
	tmplBytes, _ := os.ReadFile(filepath.Join(templatesRoot, "index.html"))
	html := strings.ReplaceAll(string(tmplBytes), "{{TITLE}}", escapeHTML(*title))
	bb, _ := json.MarshalIndent(out, "", "  ")
	html = strings.ReplaceAll(html, "{{INLINE_JSON}}", string(bb))
	if err := os.WriteFile(filepath.Join(*outDir, "index.html"), []byte(html), 0o644); err != nil {
		panic(err)
	}

	fmt.Println("Done. Open:", filepath.Join(*outDir, "index.html"))
}

// ---------------- CSV parsing ----------------
func parseCSV(path string) ([]Record, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	headerLine, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	// detect delimiter
	delim := ','
	if strings.Count(headerLine, ";") > strings.Count(headerLine, ",") {
		delim = ';'
	} else if strings.Contains(headerLine, "\t") {
		delim = '\t'
	}

	r := csv.NewReader(io.MultiReader(strings.NewReader(headerLine), br))
	r.Comma = delim
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("empty file")
	}

	header := rows[0]
	// Need at least 3 columns: X, Y, Value; 4th (Size) optional
	if len(header) < 3 {
		return nil, nil, fmt.Errorf("need at least 3 columns: X, Y, Value")
	}

	numRe := regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)?`)
	out := make([]Record, 0, len(rows)-1)

	for _, row := range rows[1:] {
		if len(strings.TrimSpace(strings.Join(row, ""))) == 0 {
			continue
		}
		rec := Record{Extras: map[string]string{}}
		if len(row) > 0 {
			rec.X = atoiSafe(row, 0)
		}
		if len(row) > 1 {
			rec.Y = atoiSafe(row, 1)
		}
		if len(row) > 2 {
			// empty cell - no data
			if strings.TrimSpace(row[2]) == "" {
				rec.Value = -1
			} else {
				rec.Value = atofSmart(row[2], numRe)
			}
		}
		if len(row) > 3 {
			rec.Size = atofSmart(row[3], numRe)
		}

		// extras from 5th column onwards
		if len(header) > 4 {
			for i := 4; i < len(header) && i < len(row); i++ {
				rec.Extras[strings.TrimSpace(header[i])] = strings.TrimSpace(row[i])
			}
		}
		out = append(out, rec)
	}

	return out, header, nil
}

func atoiSafe(row []string, i int) int {
	if i < 0 || i >= len(row) {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(row[i]))
	return v
}

func atofSmart(s string, re *regexp.Regexp) float64 {
	s = strings.TrimSpace(s)
	if re != nil {
		if m := re.FindString(s); m != "" {
			s = m
		}
	}
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v

}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
