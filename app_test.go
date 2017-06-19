package goblog

import (
	"testing"
	"fmt"
)

func TestNew(t *testing.T) {
	app := New()
	fmt.Println()
	fmt.Println(app)
}
