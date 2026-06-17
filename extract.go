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

package main

import (
	"crypto/cipher"
	"fmt"
	"io"
	"os"

	"github.com/patapancakes/exdepot/gozelle"
)

type ExtractorJob struct {
	Path string
	File *gozelle.File
}

func extractorWorker(path string, file *gozelle.File, data io.ReaderAt, block cipher.Block) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}

	err = file.Prepare(block, data)
	if err != nil {
		return fmt.Errorf("failed to prepare file for reading: %w", err)
	}

	_, err = io.Copy(out, file)
	if err != nil {
		return fmt.Errorf("failed to extract cache file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close cache file: %w", err)
	}

	err = out.Close()
	if err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	return nil
}
