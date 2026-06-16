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
)

type DirEntry struct {
	file File
}

func (s DirEntry) Name() string {
	return s.file.item.Name
}

func (s DirEntry) IsDir() bool {
	return s.file.item.IsDirectory()
}

func (s DirEntry) Type() fs.FileMode {
	stat, _ := s.file.Stat()
	return stat.Mode().Type()
}

func (s DirEntry) Info() (fs.FileInfo, error) {
	return s.file.Stat()
}
