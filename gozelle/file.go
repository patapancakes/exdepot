/*
	Copyright (C) 2024-2025  Pancakes <patapancakes@pagefault.games>

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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

func (f *File) Close() error {
	for _, c := range f.Chunks {
		err := c.Close()
		if err != nil {
			return err
		}
	}

	// HACK: makes it not leak a ton of memory
	f.Chunks = nil

	return nil
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
