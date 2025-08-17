package template

import (
	"time"

	"github.com/romshark/todostar/domain"
)

func dueDateOver(now, due time.Time) bool { return now.Unix() > due.Unix() }

func percentDone(s []*domain.Todo) float64 {
	if len(s) == 0 {
		return 0
	}
	var done int
	for _, t := range s {
		if t.Status == domain.StatusDone {
			done++
		}
	}
	return float64(done) / float64(len(s)) * 100
}
