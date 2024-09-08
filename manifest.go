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

package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

var sanitize = strings.NewReplacer("\\", "", "/", "", ":", "", "*", "", "\"", "", "<", "", ">", "", "|", "").Replace

type Manifest struct {
	Dummy1       uint32 // unused
	StoredAppID  uint32 // unused
	StoredAppVer uint32 // unused
	NumItems     uint32
	NumFiles     uint32 // unused
	BlockSize    uint32 // unused
	DirSize      uint32 // unused
	DirNameSize  uint32 // unused
	Info1Count   uint32 // unused
	CopyCount    uint32 // unused
	LocalCount   uint32 // unused
	Dummy2       uint32 // unused
	Dummy3       uint32 // unused
	Checksum     uint32 // unused
	Items        []Item
}

type Item struct {
	NameOffset  uint32
	Size        uint32
	ID          uint32
	Type        uint32
	ParentIndex uint32
	NextIndex   uint32
	FirstIndex  uint32
	Name        string
	Path        string // not in file
}

func (i Item) IsDirectory() bool {
	return i.Type&0x4000 == 0
}

func readManifest(manifestdir string, depot int, version int) (Manifest, error) {
	file, err := os.Open(fmt.Sprintf("%s/%d_%d.manifest", manifestdir, depot, version))
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
	manifest.StoredAppID = v[1]
	manifest.StoredAppVer = v[2]
	manifest.NumItems = v[3]
	manifest.NumFiles = v[4]
	manifest.BlockSize = v[5]
	manifest.DirSize = v[6]
	manifest.DirNameSize = v[7]
	manifest.Info1Count = v[8]
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

		item.NameOffset = v[0]
		item.Size = v[1]
		item.ID = v[2]
		item.Type = v[3]
		item.ParentIndex = v[4]
		item.NextIndex = v[5]
		item.FirstIndex = v[6]

		// name offset but no name size? really???
		_, err = r.Seek(int64(56+(manifest.NumItems*28)+item.NameOffset), 0)
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
		// manifest.Items[i] SHOULD always exist
		path := manifest.Items[i].Name

		// could probably do some fancy iterator but idk how to do that
		parent := manifest.Items[i]
		for parent.ParentIndex != 0xFFFFFFFF {
			// again, should exist
			parent = manifest.Items[parent.ParentIndex]

			path = parent.Name + "/" + path
		}

		manifest.Items[i].Path = path
	}

	return manifest, nil
}
