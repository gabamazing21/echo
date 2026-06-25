package handlers

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"consensus/internal/models"
)

// Renderer implements echo.Renderer using one html/template set per page,
// each composed of the shared base layout plus that page's template.
type Renderer struct {
	pages map[string]*template.Template
}

// NewRenderer parses base.html + every other *.html page from the template FS.
func NewRenderer(tmplFS fs.FS) (*Renderer, error) {
	funcs := templateFuncs()

	entries, err := fs.ReadDir(tmplFS, "templates")
	if err != nil {
		return nil, err
	}
	r := &Renderer{pages: map[string]*template.Template{}}
	for _, e := range entries {
		name := e.Name()
		if name == "base.html" || !strings.HasSuffix(name, ".html") {
			continue
		}
		t, err := template.New("base.html").Funcs(funcs).
			ParseFS(tmplFS, "templates/base.html", "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		r.pages[name] = t
	}
	return r, nil
}

// Render satisfies echo.Renderer. name is a page file like "today.html".
func (r *Renderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	t, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return t.ExecuteTemplate(w, "base.html", data)
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"fmtDate":  fmtDate,
		"daysTo":   daysTo,
		"dueText":  dueText,
		"dueClass": dueClass,
		"pad2":     func(i int) string { return fmt.Sprintf("%02d", i) },
		"pct":      func(f float64) string { return fmt.Sprintf("%.0f", f*100) },
		"add":      func(a, b int) int { return a + b },
		"list":     func(s ...string) []string { return s },
		"parseOpts": parseOpts,
		"safe":     func(s string) template.HTML { return template.HTML(s) }, //nolint:gosec // lesson HTML is authored by us
		"ring":     ringSVG,
		"dict":     dict,
		"firstLine": func(s string) string {
			if i := strings.IndexByte(s, '\n'); i >= 0 {
				return s[:i]
			}
			return s
		},
	}
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("02 Jan")
}

func daysTo(t time.Time) int {
	d := time.Until(t)
	return int(math.Ceil(d.Hours() / 24))
}

func dueText(t time.Time) string {
	n := daysTo(t)
	switch {
	case n < 0:
		return fmt.Sprintf("%dd overdue", -n)
	case n == 0:
		return "due today"
	case n == 1:
		return "due tomorrow"
	default:
		return fmt.Sprintf("in %dd", n)
	}
}

func dueClass(t time.Time) string {
	n := daysTo(t)
	switch {
	case n < 0:
		return "over"
	case n <= 4:
		return "soon"
	default:
		return "ok"
	}
}

// dict builds a map from alternating key/value pairs, for passing multiple
// values into a sub-template: {{template "x" (dict "A" 1 "B" 2)}}.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict needs an even number of args")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		k, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		m[k] = pairs[i+1]
	}
	return m, nil
}

// ringSVG renders the segmented progress ring, ported from the original app's
// JS. One arc segment per step; filled teal (partial) or green+glow (complete).
func ringSVG(steps []models.Step) template.HTML {
	const cx, cy, r, gap = 160.0, 160.0, 128.0, 4.0
	n := len(steps)
	if n == 0 {
		return ""
	}
	seg := 360.0 / float64(n)

	var b strings.Builder
	for i, st := range steps {
		a0 := float64(i)*seg + gap/2
		a1 := float64(i+1)*seg - gap/2
		f := st.Frac()
		full := st.Full()

		// track
		fmt.Fprintf(&b, `<path d="%s" stroke="var(--raised)" stroke-width="15" fill="none" stroke-linecap="round"/>`,
			arc(cx, cy, r, a0, a1))
		// fill
		if f > 0 {
			a1f := a0 + (a1-a0)*f
			color := "var(--teal)"
			glow := ""
			if full {
				color = "var(--green)"
				glow = ` filter="url(#glow)"`
			}
			fmt.Fprintf(&b, `<path d="%s" stroke="%s" stroke-width="15" fill="none" stroke-linecap="round"%s/>`,
				arc(cx, cy, r, a0, a1f), color, glow)
		}
		// number label at the segment's midpoint
		lx, ly := polar(cx, cy, r, (a0+a1)/2)
		labelColor := "var(--faint)"
		if full {
			labelColor = "var(--bg)"
		}
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" text-anchor="middle" font-family="var(--mono)" font-size="9" fill="%s" font-weight="600">%d</text>`,
			lx, ly+3, labelColor, st.ID)
	}
	return template.HTML(b.String()) //nolint:gosec // all values are numeric/constant
}

// Opt is one multiple-choice option parsed from a quiz's options string.
type Opt struct {
	Letter string // "A".."D"
	Full   string // e.g. "A) nil"
}

var optMarker = regexp.MustCompile(`[A-D]\)`)

// parseOpts turns "A) nil  B) []  C) struct  D) 0" into structured options.
// Go's RE2 has no lookahead, so we locate each marker and slice between them.
func parseOpts(s string) []Opt {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" {
		return nil
	}
	locs := optMarker.FindAllStringIndex(s, -1)
	opts := make([]Opt, 0, len(locs))
	for i, loc := range locs {
		end := len(s)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		opts = append(opts, Opt{
			Letter: s[loc[0] : loc[0]+1],
			Full:   strings.TrimSpace(s[loc[0]:end]),
		})
	}
	return opts
}

func polar(cx, cy, r, a float64) (float64, float64) {
	rad := (a - 90) * math.Pi / 180
	return cx + r*math.Cos(rad), cy + r*math.Sin(rad)
}

func arc(cx, cy, r, a0, a1 float64) string {
	x0, y0 := polar(cx, cy, r, a1)
	x1, y1 := polar(cx, cy, r, a0)
	big := 0
	if a1-a0 > 180 {
		big = 1
	}
	return fmt.Sprintf("M %.2f %.2f A %.2f %.2f 0 %d 0 %.2f %.2f", x0, y0, r, r, big, x1, y1)
}
