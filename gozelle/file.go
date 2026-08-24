/*
	Copyright (C) 2024-2026  Pancakes <patapancakes@pagefault.games>

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
	"crypto/cipher"
	"io"
)

type FileHeader struct {
	ID     uint64
	Length uint64
	Mode   Mode
}

type File struct {
	FileHeader

	Blocks []*Block `json:"blocks"`

	data io.Reader
}

func (f *File) Read(dst []byte) (int, error) {
	if f.data == nil {
		return 0, ErrBlockNotPrepared
	}

	return f.data.Read(dst)
}

func (f *File) Close() error {
	for _, b := range f.Blocks {
		err := b.Close()
		if err != nil {
			return err
		}
	}

	f.data = nil

	return nil
}

func (f *File) Prepare(block cipher.Block, src io.ReaderAt) error {
	for _, b := range f.Blocks {
		err := b.Prepare(block, io.NewSectionReader(src, int64(b.Offset), int64(b.Length)), f.Mode)
		if err != nil {
			return err
		}
	}

	readers := make([]io.Reader, 0, len(f.Blocks))
	for _, b := range f.Blocks {
		readers = append(readers, b)
	}

	f.data = io.MultiReader(readers...)

	return nil
}
