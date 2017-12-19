package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime/debug"
	"sync"
)

type FileStatus struct {
	Name     string
	Path     string
	Priority int

	Exists chan struct{}
}

func (f FileStatus) FileInfo() os.FileInfo {
	file, err := os.Stat(f.Path)
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
	return file
}

type tracker struct {
	sync.Mutex
	m map[string]FileStatus
}

var FileTracker = tracker{m: make(map[string]FileStatus)}

// Add takes a FileStatus as input, and outputs whether another entry existed
// previously.
//
// If the file referred to by the FileStatus exists, then it returns true.
// Otherwise, the FileStatus gets added to the map and Add returns false.
//
// The function calling Add is expected to wait on fs.Exists to determine when
// to link files if true is returned. Otherwise, if false is returned, it is
// expected that the file will be downloaded, and fs.Exists will be closed when
// the download is completed.
func (t *tracker) Add(name, path string) bool {
	t.Lock()
	defer t.Unlock()
	if _, ok := t.m[name]; ok {
		// Entry exists.
		return true
	}

	// Entry does not exist.
	t.m[name] = FileStatus{
		Name:     name,
		Path:     path,
		Priority: 0, // TODO(Liru): Add priority to file list when it is implemented
		Exists:   make(chan struct{}),
	}
	return false
}

func (t *tracker) Link(oldfilename, newpath string) {
	t.Lock()
	defer t.Unlock()
	info := t.m[oldfilename]
	newInfo := FileInfo(newpath)
	if !os.SameFile(info.FileInfo(), newInfo) {

		err := os.MkdirAll(path.Dir(newpath), 0755)
		if err != nil {
			log.Fatal(err)
		}

		os.Remove(newpath)
		err = os.Link(info.Path, newpath)
		if err != nil {
			log.Println(info.Path, " - ", newpath)
			log.Fatal("t.Link", err)
		}
	}
}

func (t *tracker) WaitForDownload(name string) {
	t.Lock()
	ch := t.m[name].Exists
	t.Unlock()
	<-ch
}

// Signal informs the goroutines waiting for a file to finish downloading that
// the file specified is now present on disk. This allows them to hardlink to
// it.
func (t *tracker) Signal(file string) {
	t.Lock()
	defer t.Unlock()
	close(t.m[file].Exists)
}

// DirectoryScanner implements filepath.WalkFunc, necessary to walk and
// register each file in the download directory before beginning the
// download. This lets us know which files are already downloaded, and
// which ones can be hardlinked.
//
// Deprecated; replaced by GetAllCurrentFiles().
func DirectoryScanner(path string, f os.FileInfo, err error) error {
	if f == nil { // Only exists if the directory doesn't exist beforehand.
		return err
	}

	if f.IsDir() {
		return err
	}

	if info, ok := FileTracker.m[f.Name()]; ok {
		// File exists.
		if !os.SameFile(info.FileInfo(), f) {
			os.Remove(path)
			err := os.Link(info.Path, path)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		// New file.
		closedChannel := make(chan struct{})
		close(closedChannel)

		FileTracker.m[f.Name()] = FileStatus{
			Name:     f.Name(),
			Path:     path,
			Priority: 0, // TODO(Liru): Add priority to file list when it is implemented
			Exists:   closedChannel,
		}

	}
	return err
}

// GetAllCurrentFiles scans the download directory and parses the files inside
// for possible future linking, if a duplicate is found.
func GetAllCurrentFiles() {
	os.MkdirAll(cfg.DownloadDirectory, 0755)
	dirs, err := ioutil.ReadDir(cfg.DownloadDirectory)
	if err != nil {
		panic(err)
	}

	// TODO: Make GetAllCurrentFiles a LOT more stable. A lot could go wrong, but meh.

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		dir, err := os.Open(cfg.DownloadDirectory + string(os.PathSeparator) + d.Name())

		if err != nil {
			log.Fatal(err)
		}
		// fmt.Println(dir.Name())
		files, err := dir.Readdirnames(0)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if info, ok := FileTracker.m[f]; ok {
				// File exists.

				p := dir.Name() + string(os.PathSeparator) + f

				checkFile, err := os.Stat(p)
				if err != nil {
					log.Fatal(err)
				}

				if !os.SameFile(info.FileInfo(), checkFile) {
					os.Remove(p)
					err := os.Link(info.Path, p)
					if err != nil {
						log.Fatal(err)
					}
				}
			} else {
				// New file.
				closedChannel := make(chan struct{})
				close(closedChannel)

				FileTracker.m[f] = FileStatus{
					Name:     f,
					Path:     dir.Name() + string(os.PathSeparator) + f,
					Priority: 0, // TODO(Liru): Add priority to file list when it is implemented
					Exists:   closedChannel,
				}

			}
		}

	}
}

func FileInfo(s string) os.FileInfo {
	file, err := os.Stat(s)
	if err != nil {
		// checkError(err, "| FileInfo(s) |")
	}
	return file
}
