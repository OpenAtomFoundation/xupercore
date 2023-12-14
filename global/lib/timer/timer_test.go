package timer

import (
	"fmt"
	"testing"
	"time"
)

func TestMark(t *testing.T) {
	tmr := NewXTimer()
	time.Sleep(1 * time.Second)
	tmr.Mark("step_1")
	time.Sleep(1 * time.Second)
	tmr.Mark("step_2")

	fmt.Println(tmr.Print())
}
