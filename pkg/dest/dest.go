package dest

import (
	"fmt"
	"github.com/swisscom/searchdump/pkg/file"
)

type Dester interface {
	String() string
	Write(file.File) error
}

var _ Dester = (*NoneDest)(nil)
type NoneDest struct {}

func (n NoneDest) Write(f file.File) error {
	return fmt.Errorf("I cannot be used to write files")
}

func (n NoneDest) String() string {
	return "none"
}

