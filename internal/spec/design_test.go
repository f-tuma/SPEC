package spec

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func luminance(hex string) float64 {
	channel := func(s string) float64 {
		v, _ := strconv.ParseUint(s, 16, 8)
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	hex = strings.TrimPrefix(hex, "#")
	return 0.2126*channel(hex[0:2]) + 0.7152*channel(hex[2:4]) + 0.0722*channel(hex[4:6])
}
func contrast(a, b string) float64 {
	x, y := luminance(a), luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

// Every starter is a usable, accessible DESIGN.md in the Google format.
func TestDesignStartersAreCompleteAndAccessible(t *testing.T) {
	starters, err := DesignStarters(os.DirFS("../../plugin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(starters) != 6 {
		t.Fatalf("expected six starters, found %d", len(starters))
	}
	sections := []string{"## Overview", "## Colors", "## Typography", "## Layout", "## Elevation & Depth", "## Shapes", "## Components", "## Do's and Don'ts", "## Personalize this starter"}
	reference := regexp.MustCompile(`\{([a-z]+)\.([A-Za-z0-9_-]+)\}`)
	hex := regexp.MustCompile(`^#[0-9A-F]{6}$`)
	for _, s := range starters {
		t.Run(s.ID, func(t *testing.T) {
			if !strings.Contains(s.Content, "version: alpha\n") || s.Overview == "" || strings.ContainsAny(s.Overview, "{}`") {
				t.Fatal("missing version or overview")
			}
			last := -1
			for _, heading := range sections {
				at := strings.Index(s.Content, "\n"+heading+"\n")
				if at < last {
					t.Fatalf("%q is missing or out of order", heading)
				}
				last = at
			}
			if !strings.Contains(s.Content, "<!-- SPEC design starter: "+s.ID+".") {
				t.Fatal("missing starter marker")
			}
			colors := map[string]string{}
			for _, c := range s.Colors {
				if !hex.MatchString(c.Value) {
					t.Fatalf("color %s is not uppercase #RRGGBB: %s", c.Name, c.Value)
				}
				colors[c.Name] = c.Value
			}
			for name, value := range colors {
				if on, ok := colors["on-"+name]; ok {
					if ratio := contrast(value, on); ratio < 4.5 {
						t.Errorf("%s/on-%s contrast %.2f is below WCAG AA", name, name, ratio)
					}
				}
			}
			for _, pair := range [][2]string{{"surface", "on-surface-muted"}, {"background", "primary"}, {"surface", "primary"}} {
				if ratio := contrast(colors[pair[0]], colors[pair[1]]); ratio < 4.5 {
					t.Errorf("%s on %s contrast %.2f is below WCAG AA", pair[1], pair[0], ratio)
				}
			}
			for _, required := range []string{"primary", "on-primary", "background", "on-background", "surface", "on-surface", "error", "on-error"} {
				if colors[required] == "" {
					t.Errorf("missing color role %s", required)
				}
			}
			front, _, _ := strings.Cut(strings.TrimPrefix(s.Content, "---\n"), "\n---\n")
			for _, match := range reference.FindAllStringSubmatch(s.Content, -1) {
				if !regexp.MustCompile(`(?m)^` + match[1] + `:\n(?:  .*\n)*?  ` + regexp.QuoteMeta(match[2]) + `:`).MatchString(front + "\n") {
					t.Errorf("reference %s points to a missing token", match[0])
				}
			}
			for _, r := range s.Content {
				if r > 127 {
					t.Fatalf("starter text must be plain ASCII: %q", r)
				}
			}
		})
	}
}

func TestCreateDesignNeverOverwritesAndPreviewDoesNotWrite(t *testing.T) {
	m, project, _ := managerFixture(t)
	plan, err := m.CreateDesign(project, "calm-minimal", true)
	if err != nil || plan["status"] != "planned" || exists(filepath.Join(project, "DESIGN.md")) {
		t.Fatal("preview wrote or failed:", plan, err)
	}
	if _, err = m.CreateDesign(project, "no-such-starter", false); err == nil {
		t.Fatal("unknown starter accepted")
	}
	result, err := m.CreateDesign(project, "calm-minimal", false)
	if err != nil || result["status"] != "created" {
		t.Fatal(result, err)
	}
	data, _ := os.ReadFile(filepath.Join(project, "DESIGN.md"))
	if !strings.HasPrefix(string(data), "---\nversion: alpha\nname: Calm Minimal\n") {
		t.Fatal("starter content was not copied")
	}
	put(t, filepath.Join(project, "DESIGN.md"), "owner design")
	if _, err = m.CreateDesign(project, "editorial", false); err == nil || !strings.Contains(err.Error(), "nothing was overwritten") {
		t.Fatal("existing DESIGN.md was not protected:", err)
	}
	if data, _ = os.ReadFile(filepath.Join(project, "DESIGN.md")); string(data) != "owner design" {
		t.Fatal("DESIGN.md changed")
	}
}
