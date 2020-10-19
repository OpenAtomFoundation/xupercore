package timer

import (
	"fmt"
	"strings"
	"time"
)

// MarkPoint define data structure of marked point
type MarkPoint struct {
	tag   string
	delta float64
}

// XTimer define the timestamps and marked points
type XTimer struct {
	bornTime   int64
	latestTime int64
	points     []*MarkPoint
}

// NewXTimer create new XTimer instance
func NewXTimer() *XTimer {
	now := time.Now().UnixNano()
	return &XTimer{
		bornTime:   now,
		latestTime: now,
	}
}

// Mark mark a point and record the tag of the point with time delta
func (timer *XTimer) Mark(tag string) {
	now := time.Now().UnixNano()
	delta := float64(now - timer.latestTime)
	point := &MarkPoint{
		tag:   tag,
		delta: delta,
	}
	timer.latestTime = now
	timer.points = append(timer.points, point)
}

// Print all record points and timestamp information
func (timer *XTimer) Print() string {
	now := time.Now().UnixNano()
	deltaTotal := float64(now - timer.bornTime)
	msg := []string{}
	for _, point := range timer.points {
		msg = append(msg, fmt.Sprintf("%s:%.2fms", point.tag, point.delta/float64(time.Millisecond)))
	}
	msg = append(msg, fmt.Sprintf("total:%.2fms", deltaTotal/float64(time.Millisecond)))
	return strings.Join(msg, ",")
}
