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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"runtime"
	"slices"
	"strings"
)

var sanitize = strings.NewReplacer("\\", "", "/", "", ":", "", "*", "", "\"", "", "<", "", ">", "", "|", "").Replace

type Manifest struct {
	Dummy1       uint32 `json:"dummy1"`
	DepotID      uint32 `json:"depotID"`
	DepotVersion uint32 `json:"depotVersion"`
	NumItems     uint32 `json:"numItems"`
	NumFiles     uint32 `json:"numFiles"`
	BlockSize    uint32 `json:"blockSize"`
	DirSize      uint32 `json:"dirSize"`
	DirNameSize  uint32 `json:"dirNameSize"`
	InfoCount    uint32 `json:"infoCount"`
	CopyCount    uint32 `json:"copyCount"`
	LocalCount   uint32 `json:"localCount"`
	Dummy2       uint32 `json:"dummy2"`
	Dummy3       uint32 `json:"dummy3"`
	Checksum     uint32 `json:"checksum"`
	Items        []Item `json:"items"`
}

type Item struct {
	Size        uint32 `json:"size"`
	ID          uint32 `json:"id"`
	Type        uint32 `json:"type"`
	ParentIndex uint32 `json:"parentIndex"`
	NextIndex   uint32 `json:"nextIndex"`
	FirstIndex  uint32 `json:"firstIndex"`
	Name        string `json:"name"`
	Path        string `json:"path"`
}

func (i Item) IsDirectory() bool {
	return i.Type&0x4000 == 0
}

func ReadManifest(r io.ReadSeeker) (Manifest, error) {
	var manifest Manifest

	err := read(r, binary.LittleEndian, &manifest.Dummy1, &manifest.DepotID,
		&manifest.DepotVersion, &manifest.NumItems, &manifest.NumFiles, &manifest.BlockSize,
		&manifest.DirSize, &manifest.DirNameSize, &manifest.InfoCount, &manifest.CopyCount,
		&manifest.LocalCount, &manifest.Dummy2, &manifest.Dummy3, &manifest.Checksum)
	if err != nil {
		return manifest, fmt.Errorf("failed to read value: %s", err)
	}

	for i := range manifest.NumItems {
		_, err = r.Seek(int64(56+(i*28)), 0)
		if err != nil {
			return manifest, fmt.Errorf("failed to seek to item: %s", err)
		}

		var nameOffset uint32
		var item Item

		err = read(r, binary.LittleEndian, &nameOffset, &item.Size, &item.ID, &item.Type, &item.ParentIndex, &item.NextIndex, &item.FirstIndex)
		if err != nil {
			return manifest, fmt.Errorf("failed to read value: %s", err)
		}

		// name offset but no name size? really???
		_, err = r.Seek(int64(56+(manifest.NumItems*28)+nameOffset), 0)
		if err != nil {
			return manifest, fmt.Errorf("failed to seek to file name: %s", err)
		}

		namebuf := make([]byte, 256)
		_, err = r.Read(namebuf)
		if err != nil {
			return manifest, fmt.Errorf("failed to read file name: %s", err)
		}

		end := bytes.Index(namebuf, []byte{0x00})
		if end == -1 {
			return manifest, fmt.Errorf("failed to read file name: couldn't find end")
		}

		item.Name = string(namebuf[:end])

		// windows doesn't allow certain characters in file names
		if runtime.GOOS == "windows" {
			item.Name = sanitize(item.Name)
		}

		manifest.Items = append(manifest.Items, item)
	}

	for i := range manifest.NumItems {
		var hierarchy []string

		for item := manifest.Items[i]; item.ParentIndex != 0xFFFFFFFF; item = manifest.Items[item.ParentIndex] {
			hierarchy = append(hierarchy, item.Name)
		}

		slices.Reverse(hierarchy)

		manifest.Items[i].Path = path.Join(hierarchy...)
	}

	return manifest, nil
}
