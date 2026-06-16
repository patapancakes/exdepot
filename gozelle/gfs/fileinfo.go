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
	"io/fs"
	"time"

	"github.com/patapancakes/exdepot/gozelle"
)

type FileInfo struct {
	item gozelle.Item
}

func (s FileInfo) Name() string {
	return s.item.Name
}

func (s FileInfo) Size() int64 {
	return int64(s.item.Size)
}

func (s FileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm // 0777

	if !s.item.IsExecutable() {
		mode &^= 0111
	}
	if s.item.IsReadOnly() {
		mode &^= 0222
	}

	if s.item.IsDirectory() {
		mode |= fs.ModeDir
	}

	return mode
}

// time information isn't stored
func (s FileInfo) ModTime() time.Time {
	return time.Unix(0, 0)
}

func (s FileInfo) IsDir() bool {
	return s.Mode().IsDir()
}

func (s FileInfo) Sys() any {
	return nil
}
