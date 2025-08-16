package util

import "time"

type Point struct {
	TS   time.Time
	Data string
}

type Sliding struct {
	points []Point
}

func (s *Sliding) Add(p Point, keep time.Duration) {
	s.points = append(s.points, p)
	cut := p.TS.Add(-keep)
	i := 0
	for ; i < len(s.points); i++ {
		if s.points[i].TS.After(cut) { break }
	}
	if i > 0 { s.points = s.points[i:] }
}
func (s *Sliding) Count() int { return len(s.points) }
func (s *Sliding) Items() []Point { return s.points }