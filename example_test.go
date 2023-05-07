package pinggy_test

import (
	"fmt"
	"testing"

	"github.com/abhimp/pinggy"
)

func TestConnection(t *testing.T) {
	l, err := pinggy.Connect()
	if err != nil {
		t.Fatalf("Test failed: %v\n", err)
	}
	fmt.Println(l.Addr())
	l.Close()
}
