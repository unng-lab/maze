package maze

import (
	"fmt"
	"math/rand"
)

type Direction int

const (
	North Direction = iota
	South
	West
	East
)

type Walls struct {
	North bool
	South bool
	West  bool
	East  bool
}

type Cell struct {
	Visited bool
	Walls   Walls
}

type Maze struct {
	Width  int
	Height int
	Cells  [][]Cell
}

func New(width, height int) *Maze {
	cells := make([][]Cell, height)
	for y := 0; y < height; y++ {
		row := make([]Cell, width)
		for x := range row {
			row[x].Walls = Walls{North: true, South: true, West: true, East: true}
		}
		cells[y] = row
	}

	return &Maze{Width: width, Height: height, Cells: cells}
}

type neighbor struct {
	x   int
	y   int
	dir Direction
}

func (m *Maze) neighbors(x, y int) []neighbor {
	options := make([]neighbor, 0, 4)
	if y > 0 {
		options = append(options, neighbor{x: x, y: y - 1, dir: North})
	}
	if y < m.Height-1 {
		options = append(options, neighbor{x: x, y: y + 1, dir: South})
	}
	if x > 0 {
		options = append(options, neighbor{x: x - 1, y: y, dir: West})
	}
	if x < m.Width-1 {
		options = append(options, neighbor{x: x + 1, y: y, dir: East})
	}

	return options
}

func (m *Maze) CarvePassages(rng *rand.Rand) {
	type stackEntry struct {
		x int
		y int
	}

	stack := []stackEntry{{x: 0, y: 0}}
	m.Cells[0][0].Visited = true

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		neighbors := make([]neighbor, 0, 4)
		for _, option := range m.neighbors(current.x, current.y) {
			if !m.Cells[option.y][option.x].Visited {
				neighbors = append(neighbors, option)
			}
		}

		if len(neighbors) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}

		nextIndex := rng.Intn(len(neighbors))
		next := neighbors[nextIndex]

		m.setWall(current.x, current.y, next.dir, false)
		m.setWall(next.x, next.y, opposite(next.dir), false)

		m.Cells[next.y][next.x].Visited = true
		stack = append(stack, stackEntry{x: next.x, y: next.y})
	}
}

func opposite(dir Direction) Direction {
	switch dir {
	case North:
		return South
	case South:
		return North
	case West:
		return East
	case East:
		return West
	default:
		return North
	}
}

func (m *Maze) setWall(x, y int, dir Direction, value bool) {
	walls := &m.Cells[y][x].Walls
	switch dir {
	case North:
		walls.North = value
	case South:
		walls.South = value
	case West:
		walls.West = value
	case East:
		walls.East = value
	}
}

func (m *Maze) hasWall(x, y int, dir Direction) bool {
	walls := m.Cells[y][x].Walls
	switch dir {
	case North:
		return walls.North
	case South:
		return walls.South
	case West:
		return walls.West
	case East:
		return walls.East
	default:
		return true
	}
}

type Exit struct {
	X     int
	Y     int
	Dir   Direction
	Label string
}

func (m *Maze) CreateExits(labels []string, rng *rand.Rand) []Exit {
	candidates := make([]Exit, 0, 2*(m.Width+m.Height))

	for x := 0; x < m.Width; x++ {
		candidates = append(candidates, Exit{X: x, Y: 0, Dir: North})
		candidates = append(candidates, Exit{X: x, Y: m.Height - 1, Dir: South})
	}

	for y := 0; y < m.Height; y++ {
		candidates = append(candidates, Exit{X: 0, Y: y, Dir: West})
		candidates = append(candidates, Exit{X: m.Width - 1, Y: y, Dir: East})
	}

	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	exits := make([]Exit, 0, len(labels))
	used := make(map[[2]int]bool)

	for _, candidate := range candidates {
		if len(exits) == len(labels) {
			break
		}

		key := [2]int{candidate.X, candidate.Y}
		if used[key] {
			continue
		}

		candidate.Label = labels[len(exits)]
		used[key] = true
		exits = append(exits, candidate)
		m.setWall(candidate.X, candidate.Y, candidate.Dir, false)
	}

	return exits
}

type Point struct {
	X int
	Y int
}

func (m *Maze) SolvePath(start, goal Exit) ([]Point, error) {
	startPoint := Point{X: start.X, Y: start.Y}
	goalPoint := Point{X: goal.X, Y: goal.Y}

	queue := []Point{startPoint}
	visited := map[Point]bool{startPoint: true}
	previous := make(map[Point]Point)

	directions := []struct {
		dir Direction
		dx  int
		dy  int
	}{
		{dir: North, dx: 0, dy: -1},
		{dir: South, dx: 0, dy: 1},
		{dir: West, dx: -1, dy: 0},
		{dir: East, dx: 1, dy: 0},
	}

	found := false

	for len(queue) > 0 && !found {
		current := queue[0]
		queue = queue[1:]

		if current == goalPoint {
			found = true
			break
		}

		for _, d := range directions {
			if m.hasWall(current.X, current.Y, d.dir) {
				continue
			}

			next := Point{X: current.X + d.dx, Y: current.Y + d.dy}
			if next.X < 0 || next.X >= m.Width || next.Y < 0 || next.Y >= m.Height {
				continue
			}

			if visited[next] {
				continue
			}

			visited[next] = true
			previous[next] = current
			queue = append(queue, next)
		}
	}

	if !found {
		return nil, fmt.Errorf("no path between exits %s and %s", start.Label, goal.Label)
	}

	path := []Point{goalPoint}

	for path[len(path)-1] != startPoint {
		path = append(path, previous[path[len(path)-1]])
	}

	for i := 0; i < len(path)/2; i++ {
		oppositeIndex := len(path) - 1 - i
		path[i], path[oppositeIndex] = path[oppositeIndex], path[i]
	}

	return path, nil
}
