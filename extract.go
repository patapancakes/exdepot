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
	"io"
	"log"
	"os"
	"sync"

	"github.com/patapancakes/exdepot/gozelle"
)

type ExtractorJob struct {
	Path string
	File *gozelle.File
}

func extractorWorker(wg *sync.WaitGroup, jobs chan ExtractorJob, data io.ReaderAt, key []byte) {
	defer wg.Done()

	for {
		job, ok := <-jobs
		if !ok {
			break
		}

		out, err := os.OpenFile(job.Path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("failed to open output file: %s", err)
		}

		err = job.File.Prepare(key, data)
		if err != nil {
			log.Fatalf("failed to prepare file for reading: %s", err)
		}

		_, err = io.Copy(out, job.File)
		if err != nil {
			log.Fatalf("failed to extract cache file: %s", err)
		}

		err = out.Sync()
		if err != nil {
			log.Fatalf("failed to sync output file: %s", err)
		}

		err = out.Close()
		if err != nil {
			log.Fatalf("failed to close output file: %s", err)
		}
	}
}
