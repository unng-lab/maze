package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
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

type point struct {
	x int
	y int
}

func (m *maze) solvePath(start, goal exit) ([]point, error) {
	startCell := point{x: start.x, y: start.y}
	goalCell := point{x: goal.x, y: goal.y}

	queue := []point{startCell}
	visited := map[point]bool{startCell: true}
	prev := map[point]point{}

	directions := []struct {
		dir direction
		dx  int
		dy  int
	}{
		{north, 0, -1},
		{south, 0, 1},
		{west, -1, 0},
		{east, 1, 0},
	}

	var found bool
	for len(queue) > 0 && !found {
		current := queue[0]
		queue = queue[1:]

		if current == goalCell {
			found = true
			break
		}

		cell := m.cells[current.y][current.x]
		for _, d := range directions {
			if cell.walls[d.dir] {
				continue
			}
			nx, ny := current.x+d.dx, current.y+d.dy
			if nx < 0 || nx >= m.width || ny < 0 || ny >= m.height {
				continue
			}
			next := point{x: nx, y: ny}
			if visited[next] {
				continue
			}
			visited[next] = true
			prev[next] = current
			queue = append(queue, next)
		}
	}

	if !found {
		return nil, fmt.Errorf("no path found between exits %s and %s", start.label, goal.label)
	}

	path := []point{goalCell}
	for path[len(path)-1] != startCell {
		path = append(path, prev[path[len(path)-1]])
	}

	for i := 0; i < len(path)/2; i++ {
		path[i], path[len(path)-1-i] = path[len(path)-1-i], path[i]
	}

	return path, nil
}

func generateHTML(m *maze, exits []exit, solution []point, cellSize int, filename string) error {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n<title>Maze</title>\n<style>\n")
	sb.WriteString("body{font-family:Arial,Helvetica,sans-serif;background:#f5f5f5;color:#222;padding:20px;}\n")
	sb.WriteString(".maze{display:grid;grid-template-columns:repeat(")
	sb.WriteString(fmt.Sprintf("%d,%dpx);gap:0;}", m.width, cellSize))
	sb.WriteString("\n.cell{width:")
	sb.WriteString(fmt.Sprintf("%dpx;height:%dpx;box-sizing:border-box;position:relative;background:#fff;}", cellSize, cellSize))
	sb.WriteString("\n.cell.top{border-top:2px solid #000;}\n.cell.right{border-right:2px solid #000;}\n.cell.bottom{border-bottom:2px solid #000;}\n.cell.left{border-left:2px solid #000;}\n")
	sb.WriteString(".cell.exit::after{content:attr(data-label);position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);font-weight:bold;color:#1565c0;}\n")
	sb.WriteString(".cell.solution{background:#fff59d;}\n")
	sb.WriteString(".legend{margin-top:20px;}\n.legend span{display:inline-block;margin-right:15px;}\n")
	sb.WriteString("</style>\n</head>\n<body>\n<h1>Maze</h1>\n")

	sb.WriteString("<div class=\"maze\">\n")

	solutionSet := map[point]bool{}
	for _, p := range solution {
		solutionSet[p] = true
	}

	exitLabels := map[point]string{}
	for _, ex := range exits {
		exitLabels[point{x: ex.x, y: ex.y}] = ex.label
	}

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			cell := m.cells[y][x]
			classes := []string{"cell"}
			if cell.walls[north] {
				classes = append(classes, "top")
			}
			if cell.walls[east] {
				classes = append(classes, "right")
			}
			if cell.walls[south] {
				classes = append(classes, "bottom")
			}
			if cell.walls[west] {
				classes = append(classes, "left")
			}
			if solutionSet[point{x: x, y: y}] {
				classes = append(classes, "solution")
			}

			label := exitLabels[point{x: x, y: y}]
			attributes := fmt.Sprintf("class=\"%s\"", strings.Join(classes, " "))
			if label != "" {
				attributes += fmt.Sprintf(" data-label=\"%s\"", label)
			}

			sb.WriteString("<div ")
			sb.WriteString(attributes)
			sb.WriteString("></div>\n")
		}
	}

	sb.WriteString("</div>\n")
	sb.WriteString("<div class=\"legend\"><span><strong>Exits:</strong> A, B, C, D</span><span><strong>Highlighted path:</strong> solution between exits A and B</span></div>\n")
	sb.WriteString("</body>\n</html>")

	return os.WriteFile(filename, []byte(sb.String()), 0o644)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	const (
		mazeWidth  = 40
		mazeHeight = 40
		cellSize   = 20
		outputFile = "maze.html"
	)

	m := newMaze(mazeWidth, mazeHeight)
	m.carvePassages()
	exits := m.createExits()

	if len(exits) < 2 {
		panic("not enough exits to compute a solution")
	}

	solution, err := m.solvePath(exits[0], exits[1])
	if err != nil {
		panic(err)
	}

	if err := generateHTML(m, exits, solution, cellSize, outputFile); err != nil {
		panic(err)
	}
}
