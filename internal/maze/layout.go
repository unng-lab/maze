package maze

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

type Layout struct {
	Maze         *Maze
	Exits        []Exit
	Solution     []Point
	SolutionFrom string
	SolutionTo   string
}

func BuildLayout(width, height int, exitLabels []string, rng *rand.Rand) (*Layout, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("maze dimensions must be positive")
	}

	if len(exitLabels) < 2 {
		return nil, errors.New("at least two exit labels are required")
	}

	generator := rng
	if generator == nil {
		generator = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	m := New(width, height)
	m.CarvePassages(generator)
	exits := m.CreateExits(exitLabels, generator)

	startLabel := exitLabels[0]
	endLabel := exitLabels[1]

	startExit, ok := findExit(exits, startLabel)
	if !ok {
		return nil, errors.New("start exit not found")
	}

	endExit, ok := findExit(exits, endLabel)
	if !ok {
		return nil, errors.New("end exit not found")
	}

	solution, err := m.SolvePath(startExit, endExit)
	if err != nil {
		return nil, err
	}

	return &Layout{
		Maze:         m,
		Exits:        exits,
		Solution:     solution,
		SolutionFrom: startExit.Label,
		SolutionTo:   endExit.Label,
	}, nil
}

func findExit(exits []Exit, label string) (Exit, bool) {
	for _, ex := range exits {
		if ex.Label == label {
			return ex, true
		}
	}

	return Exit{}, false
}

func (l *Layout) ExitLabels() []string {
	labels := make([]string, len(l.Exits))
	for i, ex := range l.Exits {
		labels[i] = ex.Label
	}
	return labels
}

func (l *Layout) ExitLabelMap() map[Point]string {
	result := make(map[Point]string, len(l.Exits))
	for _, ex := range l.Exits {
		result[Point{X: ex.X, Y: ex.Y}] = ex.Label
	}
	return result
}

func (l *Layout) SolutionSet() map[Point]struct{} {
	result := make(map[Point]struct{}, len(l.Solution))
	for _, point := range l.Solution {
		result[point] = struct{}{}
	}
	return result
}

type Builder interface {
	Build(width, height int, exitLabels []string) (*Layout, error)
}

type RandomBuilder struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func NewRandomBuilder(seed int64) *RandomBuilder {
	return &RandomBuilder{rng: rand.New(rand.NewSource(seed))}
}

func (b *RandomBuilder) Build(width, height int, exitLabels []string) (*Layout, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return BuildLayout(width, height, exitLabels, b.rng)
}
