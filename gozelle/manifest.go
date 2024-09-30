/*
	Copyright (C) 2024  Pancakes <patapancakes@pagefault.games>

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
	"fmt"
	"io"
	"os"
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

func ManifestFromFile(manifestdir string, depot int, version int) (Manifest, error) {
	file, err := os.Open(path.Join(manifestdir, fmt.Sprintf("%d_%d.manifest", depot, version)))
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to open manifest file: %s", err)
	}

	defer file.Close()

	manifest, err := manifestFromReader(file)
	if err != nil {
		return manifest, fmt.Errorf("failed to read manifest: %s", err)
	}

	return manifest, nil
}

func manifestFromReader(r io.ReadSeeker) (Manifest, error) {
	var manifest Manifest

	v, err := readUint32List(r, 14)
	if err != nil {
		return manifest, fmt.Errorf("failed to read value: %s", err)
	}

	manifest.Dummy1 = v[0]
	manifest.DepotID = v[1]
	manifest.DepotVersion = v[2]
	manifest.NumItems = v[3]
	manifest.NumFiles = v[4]
	manifest.BlockSize = v[5]
	manifest.DirSize = v[6]
	manifest.DirNameSize = v[7]
	manifest.InfoCount = v[8]
	manifest.CopyCount = v[9]
	manifest.LocalCount = v[10]
	manifest.Dummy2 = v[11]
	manifest.Dummy3 = v[12]
	manifest.Checksum = v[13]

	for i := range manifest.NumItems {
		_, err = r.Seek(int64(56+(i*28)), 0)
		if err != nil {
			return manifest, fmt.Errorf("failed to seek to item: %s", err)
		}

		var item Item

		v, err := readUint32List(r, 7)
		if err != nil {
			return manifest, fmt.Errorf("failed to read value: %s", err)
		}

		nameOffset := v[0]
		item.Size = v[1]
		item.ID = v[2]
		item.Type = v[3]
		item.ParentIndex = v[4]
		item.NextIndex = v[5]
		item.FirstIndex = v[6]

		// name offset but no name size? really???
		_, err = r.Seek(int64(56+(manifest.NumItems*28)+nameOffset), 0)
		if err != nil {
			return manifest, fmt.Errorf("failed to seek to file name: %s", err)
		}

		var namebuf []byte
		for i := 0; i < 256; i++ {
			b := make([]byte, 1)
			_, err = r.Read(b)
			if err != nil {
				return manifest, fmt.Errorf("failed to read file name: %s", err)
			}

			// look for null terminator
			if b[0] == 0x00 {
				break
			}

			namebuf = append(namebuf, b...)
		}

		item.Name = string(namebuf)

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
