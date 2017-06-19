package goblog

import "testing"

func TestError_From(t *testing.T) {
	var err Error
	err = Error{}
	var e error
	e = nil
	err.From(e)
}