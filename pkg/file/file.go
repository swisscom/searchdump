package file

import "io"

type File struct {
	Name    string
	Content io.Reader
}
