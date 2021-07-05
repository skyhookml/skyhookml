package skyhook

import (
	"testing"
)

func TestFloorDiv(t *testing.T) {
	check := func(a int, b int, expected int) {
		res := FloorDiv(a, b)
		if res != expected {
			t.Errorf("FloorDiv(%d, %d) = %d; want %d", a, b, res, expected)
		}
	}
	check(1, 2, 0)
	check(-1, 2, -1)
	check(-2, 2, -1)
	check(1, -2, -1)
	check(2, -2, -1)
	check(-25, 4, -7)
	check(25, 4, 6)
}
