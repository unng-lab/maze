package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"maze/internal/maze"
)

//go:embed templates/maze.html
var templatesFS embed.FS

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer() (*Renderer, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/maze.html")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &Renderer{tmpl: tmpl}, nil
}

type templateCell struct {
	Top        bool
	Right      bool
	Bottom     bool
	Left       bool
	IsSolution bool
	Label      string
}

type templateData struct {
	Width         int
	Height        int
	CellSize      int
	Cells         [][]templateCell
	ExitLabels    []string
	SolutionRange string
}

func (r *Renderer) Render(layout *maze.Layout, cellSize int) (string, error) {
	exitLabels := layout.ExitLabels()
	exitLabelMap := layout.ExitLabelMap()
	solutionSet := layout.SolutionSet()

	cells := make([][]templateCell, layout.Maze.Height)
	for y := 0; y < layout.Maze.Height; y++ {
		row := make([]templateCell, layout.Maze.Width)
		for x := 0; x < layout.Maze.Width; x++ {
			cell := layout.Maze.Cells[y][x]
			point := maze.Point{X: x, Y: y}

			_, hasSolution := solutionSet[point]
			label := exitLabelMap[point]

			row[x] = templateCell{
				Top:        cell.Walls.North,
				Right:      cell.Walls.East,
				Bottom:     cell.Walls.South,
				Left:       cell.Walls.West,
				IsSolution: hasSolution,
				Label:      label,
			}
		}
		cells[y] = row
	}

	data := templateData{
		Width:         layout.Maze.Width,
		Height:        layout.Maze.Height,
		CellSize:      cellSize,
		Cells:         cells,
		ExitLabels:    exitLabels,
		SolutionRange: fmt.Sprintf("%s → %s", layout.SolutionFrom, layout.SolutionTo),
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
