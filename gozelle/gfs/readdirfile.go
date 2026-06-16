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

package gfs

import (
	"io"
	"io/fs"

	"github.com/patapancakes/exdepot/gozelle"
)

type ReadDirFile struct {
	File

	items  []gozelle.Item
	offset int
}

func (s ReadDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	length := len(s.items) - s.offset
	if n > 0 {
		length = min(length, n)
	}

	entries := make([]fs.DirEntry, 0, length)
	for _, i := range s.items[s.offset : s.offset+length] {
		entries = append(entries, DirEntry{file: File{item: i}})
	}

	s.offset += length

	if n > 0 && s.offset >= len(s.items) {
		return entries, io.EOF
	}

	return entries, nil
}

func (s ReadDirFile) Read(dst []byte) (int, error) {
	return 0, io.EOF
}

func (s ReadDirFile) Close() error {
	return nil
}
