package mbvip

import "fmt"

var (
	ErrUnsupportedPlatform = fmt.Errorf("mbvip only supports windows")
)

type ProcNotFoundError struct {
	Name string
}

func (e ProcNotFoundError) Error() string {
	return fmt.Sprintf("required api not found: %s", e.Name)
}
