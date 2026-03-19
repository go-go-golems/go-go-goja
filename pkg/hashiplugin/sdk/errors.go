package sdk

import "fmt"

func argIndexError(index int, size int) error {
	return fmt.Errorf("sdk call argument %d out of range (have %d)", index, size)
}

func argTypeError(index int, want string, got any) error {
	return fmt.Errorf("sdk call argument %d expected %s, got %T", index, want, got)
}
