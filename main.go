package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"time"
)

type direction int

const (
	north direction = iota
	south
	west
	east
)

type cell struct {
	visited bool
	walls   [4]bool
}

type maze struct {
	width, height int
	cells         [][]cell
}

func newMaze(width, height int) *maze {
	m := &maze{width: width, height: height, cells: make([][]cell, height)}
	for y := 0; y < height; y++ {
		row := make([]cell, width)
		for x := 0; x < width; x++ {
			row[x].walls = [4]bool{true, true, true, true}
		}
		m.cells[y] = row
	}
	return m
}

func (m *maze) neighbors(x, y int) [][3]int {
	var neigh [][3]int
	if y > 0 {
		neigh = append(neigh, [3]int{x, y - 1, int(north)})
	}
	if y < m.height-1 {
		neigh = append(neigh, [3]int{x, y + 1, int(south)})
	}
	if x > 0 {
		neigh = append(neigh, [3]int{x - 1, y, int(west)})
	}
	if x < m.width-1 {
		neigh = append(neigh, [3]int{x + 1, y, int(east)})
	}
	return neigh
}

func opposite(dir direction) direction {
	switch dir {
	case north:
		return south
	case south:
		return north
	case west:
		return east
	case east:
		return west
	}
	return north
}

func (m *maze) carvePassages() {
	type stackEntry struct{ x, y int }
	stack := []stackEntry{{0, 0}}
	m.cells[0][0].visited = true

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		x, y := current.x, current.y
		unvisited := make([][3]int, 0)
		for _, n := range m.neighbors(x, y) {
			nx, ny := n[0], n[1]
			if !m.cells[ny][nx].visited {
				unvisited = append(unvisited, n)
			}
		}
		if len(unvisited) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		nextIdx := rand.Intn(len(unvisited))
		nx, ny, dir := unvisited[nextIdx][0], unvisited[nextIdx][1], direction(unvisited[nextIdx][2])
		m.cells[y][x].walls[dir] = false
		opp := opposite(dir)
		m.cells[ny][nx].walls[opp] = false
		m.cells[ny][nx].visited = true
		stack = append(stack, stackEntry{nx, ny})
	}
}

type exit struct {
	x, y  int
	dir   direction
	label string
}

func (m *maze) createExits() []exit {
	candidates := make([]exit, 0, 2*(m.width+m.height))
	for x := 0; x < m.width; x++ {
		candidates = append(candidates, exit{x: x, y: 0, dir: north})
		candidates = append(candidates, exit{x: x, y: m.height - 1, dir: south})
	}
	for y := 0; y < m.height; y++ {
		candidates = append(candidates, exit{x: 0, y: y, dir: west})
		candidates = append(candidates, exit{x: m.width - 1, y: y, dir: east})
	}
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	labels := []string{"A", "B", "C", "D"}
	exits := make([]exit, 0, len(labels))
	used := map[[2]int]bool{}
	for _, cand := range candidates {
		if len(exits) == len(labels) {
			break
		}
		key := [2]int{cand.x, cand.y}
		if used[key] {
			continue
		}
		exits = append(exits, exit{x: cand.x, y: cand.y, dir: cand.dir, label: labels[len(exits)]})
		used[key] = true
	}
	for _, ex := range exits {
		m.cells[ex.y][ex.x].walls[ex.dir] = false
	}
	return exits
}

var letterPatterns = map[string][]string{
	"A": {
		" XXX ",
		"X   X",
		"X   X",
		"XXXXX",
		"X   X",
		"X   X",
		"X   X",
	},
	"B": {
		"XXXX ",
		"X   X",
		"X   X",
		"XXXX ",
		"X   X",
		"X   X",
		"XXXX ",
	},
	"C": {
		" XXXX",
		"X    ",
		"X    ",
		"X    ",
		"X    ",
		"X    ",
		" XXXX",
	},
	"D": {
		"XXXX ",
		"X   X",
		"X   X",
		"X   X",
		"X   X",
		"X   X",
		"XXXX ",
	},
}

func inBounds(img image.Image, x, y int) bool {
	b := img.Bounds()
	return x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y
}

func drawLetter(img *image.RGBA, letter string, centerX, centerY, pixelSize int, col color.Color) {
	pattern, ok := letterPatterns[letter]
	if !ok {
		return
	}
	height := len(pattern)
	width := len(pattern[0])
	startX := centerX - (width*pixelSize)/2
	startY := centerY - (height*pixelSize)/2

	for y, row := range pattern {
		for x, ch := range row {
			if ch != 'X' {
				continue
			}
			for py := 0; py < pixelSize; py++ {
				for px := 0; px < pixelSize; px++ {
					ix := startX + x*pixelSize + px
					iy := startY + y*pixelSize + py
					if inBounds(img, ix, iy) {
						img.Set(ix, iy, col)
					}
				}
			}
		}
	}
}

func drawMaze(m *maze, exits []exit, cellSize, wallThickness int, filename string) error {
	width := m.width*cellSize + wallThickness
	height := m.height*cellSize + wallThickness
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	wallColor := color.Black

	drawWall := func(x0, y0, x1, y1 int) {
		if x0 == x1 {
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			for x := x0; x < x0+wallThickness; x++ {
				for y := y0; y <= y1; y++ {
					if inBounds(img, x, y) {
						img.Set(x, y, wallColor)
					}
				}
			}
		} else if y0 == y1 {
			if x0 > x1 {
				x0, x1 = x1, x0
			}
			for y := y0; y < y0+wallThickness; y++ {
				for x := x0; x <= x1; x++ {
					if inBounds(img, x, y) {
						img.Set(x, y, wallColor)
					}
				}
			}
		}
	}

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			cell := m.cells[y][x]
			px := x * cellSize
			py := y * cellSize
			if cell.walls[north] {
				drawWall(px, py, px+cellSize, py)
			}
			if cell.walls[west] {
				drawWall(px, py, px, py+cellSize)
			}
			if y == m.height-1 && cell.walls[south] {
				drawWall(px, py+cellSize, px+cellSize, py+cellSize)
			}
			if x == m.width-1 && cell.walls[east] {
				drawWall(px+cellSize, py, px+cellSize, py+cellSize)
			}
		}
	}

	letterColor := color.RGBA{0, 0, 255, 255}
	letterSize := cellSize / 4
	if letterSize < 2 {
		letterSize = 2
	}

	for _, ex := range exits {
		centerX := ex.x*cellSize + cellSize/2
		centerY := ex.y*cellSize + cellSize/2
		offset := cellSize / 3
		switch ex.dir {
		case north:
			centerY -= offset
		case south:
			centerY += offset
		case west:
			centerX -= offset
		case east:
			centerX += offset
		}
		drawLetter(img, ex.label, centerX, centerY, letterSize, letterColor)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	const (
		mazeWidth     = 40
		mazeHeight    = 40
		cellSize      = 16
		wallThickness = 3
		outputFile    = "maze.png"
	)

	m := newMaze(mazeWidth, mazeHeight)
	m.carvePassages()
	exits := m.createExits()

	if err := drawMaze(m, exits, cellSize, wallThickness, outputFile); err != nil {
		panic(err)
	}
}
