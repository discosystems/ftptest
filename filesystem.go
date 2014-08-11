package ftptest

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Filesystem struct {
	Directories []string
	Files       map[string]*File
	Mutex       sync.RWMutex
}

type File struct {
	Name         string
	Type         string
	TimeModified time.Time
	Size         int
	Content      []byte
}

// File modes
var (
	FileMode os.FileMode = 644
	DirMode  os.FileMode = 755
)

// Create a new directory
func (f *Filesystem) MkDir(path string) error {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	if err := f.DirExists(path); err == nil {
		return ErrAlreadyExists
	}

	parent := filepath.Dir(path)
	parents := strings.Split(parent, "/")

	for i, _ := range parents {
		path := strings.Join(parents[:i-1], "/")

		if err := f.DirExists(path); err != nil {
			return ErrNotFound
		}
	}

	f.Directories = append(f.Directories, filepath.Clean(path))
	return nil
}

// Recursively remove a directory
func (f *Filesystem) RmDir(path string) error {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	if err := f.DirExists(path); err != nil {
		return ErrNotFound
	}

	for name, _ := range f.Files {
		if strings.HasPrefix(name, filepath.Clean(path)) {
			delete(f.Files, name)
		}
	}

	for i, directory := range f.Directories {
		if strings.HasPrefix(directory, filepath.Clean(path)) {
			f.Directories = f.Directories[:i+copy(f.Directories[i:], f.Directories[i+1:])]
		}
	}

	return nil
}

// Write a file
func (f *Filesystem) WriteFile(path string, data []byte) error {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	dir := filepath.Dir(path)

	if err := f.DirExists(dir); err != nil {
		return ErrNoParent
	}

	f.Files[path] = &File{
		Type:         "file",
		Name:         filepath.Base(path),
		TimeModified: time.Now(),
		Size:         len(data),
		Content:      data,
	}

	return nil
}

// Read a file
func (f *Filesystem) ReadFile(path string) (*File, error) {
	f.Mutex.RLock()
	defer f.Mutex.RUnlock()

	file, exists := f.Files[path]
	if !exists {
		return nil, ErrNotFound
	}

	return file, nil
}

// Read file size
func (f *Filesystem) Size(path string) (int, error) {
	f.Mutex.RLock()
	defer f.Mutex.RUnlock()

	file, exists := f.Files[path]
	if !exists {
		return 0, ErrNotFound
	}

	return file.Size, nil
}

// Read last modified
func (f *Filesystem) LastModified(path string) (time.Time, error) {
	f.Mutex.RLock()
	defer f.Mutex.RUnlock()

	file, exists := f.Files[path]
	if !exists {
		return time.Time{}, ErrNotFound
	}

	return file.TimeModified, nil
}

// Directory exists?
func (f *Filesystem) DirExists(path string) error {
	f.Mutex.RLock()
	defer f.Mutex.RUnlock()

	for _, directory := range f.Directories {
		if directory == path {
			return nil
		}
	}

	return ErrNotFound
}

// Remove a file
func (f *Filesystem) Remove(path string) error {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	_, exists := f.Files[path]
	if !exists {
		return ErrNotFound
	}

	delete(f.Files, path)
	return nil
}

// Directory contents
func (f *Filesystem) DirContents(path string) ([]*File, error) {
	f.Mutex.RLock()
	defer f.Mutex.RUnlock()

	if err := f.DirExists(path); err != nil {
		return nil, ErrNotFound
	}

	response := make([]*File, 0)
	for name, file := range f.Files {
		if strings.HasPrefix(name, path) {
			if parts := strings.Split(name[len(path):], "/"); len(parts) == 1 {
				response = append(response, file)
			}
		}
	}

	for _, directory := range f.Directories {
		if strings.HasPrefix(directory, path) {
			if parts := strings.Split(directory[len(path):], "/"); len(parts) == 1 {
				response = append(response, &File{
					Type: "directory",
					Name: filepath.Base(path),
				})
			}
		}
	}

	return response, nil
}

// File renaming
func (f *Filesystem) Rename(from string, to string) error {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	dir := filepath.Dir(to)
	if err := f.DirExists(dir); err != nil {
		return ErrNotFound
	}

	file, err := f.ReadFile(from)
	if err != nil {
		return err
	}

	err = f.Remove(from)
	if err != nil {
		return err
	}

	return f.WriteFile(to, file.Content)
}

// Returns a mode string from the os module
func (f *File) ModeString() string {
	if f.Type == "directory" {
		return DirMode.String()
	}

	return FileMode.String()
}
