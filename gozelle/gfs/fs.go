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
	"crypto/cipher"
	"io"
	"io/fs"

	"github.com/patapancakes/exdepot/gozelle"
)

type FS struct {
	manifest gozelle.Manifest
	index    gozelle.Index

	block cipher.Block
	data  io.ReaderAt
}

func NewFS(manfest gozelle.Manifest, index gozelle.Index, block cipher.Block, data io.ReaderAt) (FS, error) {
	return FS{manifest: manfest, index: index, block: block, data: data}, nil
}

func (s FS) Open(name string) (fs.File, error) {
	if name == "." {
		name = ""
	}

	var item gozelle.Item
	var found bool
	for _, i := range s.manifest.Items {
		if i.Path != name {
			continue
		}

		item = i
		found = true
		break
	}
	if !found {
		return nil, fs.ErrNotExist
	}

	var f fs.File
	if item.IsDirectory() {
		df := ReadDirFile{File: File{item: item}}

		if item.Size > 0 {
			df.items = make([]gozelle.Item, 0, item.Size)

			for i := s.manifest.Items[item.FirstIndex]; ; i = s.manifest.Items[i.NextIndex] {
				df.items = append(df.items, i)

				if i.NextIndex == 0 {
					break
				}
			}
		}

		f = df
	} else {
		file := s.index[uint64(item.ID)]
		err := file.Prepare(s.block, s.data)
		if err != nil {
			return nil, err
		}

		f = File{item: item, file: file}
	}

	return f, nil
}
