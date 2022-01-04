package source

import "github.com/swisscom/searchdump/pkg/file"

type Sourcer interface {
	String() string
	Fetch() (chan file.File, error)
}
