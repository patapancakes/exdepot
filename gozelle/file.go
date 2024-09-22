package gozelle

import (
	"io"
)

type File struct {
	Chunks []Chunk `json:"chunks"`
	Mode   Mode    `json:"mode"`
}

func (f *File) Read(dst []byte) (int, error) {
	var readers []io.Reader
	for _, c := range f.Chunks {
		readers = append(readers, c)
	}

	return io.MultiReader(readers...).Read(dst)
}

func (f *File) Prepare(key []byte, src io.ReaderAt) error {
	for i := range f.Chunks {
		err := f.Chunks[i].Prepare(key, src, f.Mode)
		if err != nil {
			return err
		}
	}

	return nil
}
