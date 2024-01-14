package util

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"time"
)

type mapFS struct {
	files map[string][]byte
}

/*
Create fs.FS object based on a map. This is not working for the time being
*/
func NewMapFS(p map[string][]byte) *mapFS {
	if p == nil {
		p = make(map[string][]byte)
	}
	return &mapFS{
		files: p,
	}
}

func (m *mapFS) Create(name string) (fs.File, error) {
	file := &mapFile{name: name, data: []byte{}}
	m.files[name] = file.data
	return file, nil
}

func (m *mapFS) Open(name string) (fs.File, error) {
	// log.Println("Request for: ", name)
	data, ok := m.files[name]
	if !ok {
		log.Println(name, "does not exists: ")
		return nil, fs.ErrNotExist
	}
	return &mapFile{name: name, data: data}, nil
}

func (m *mapFS) Remove(name string) error {
	_, ok := m.files[name]
	if !ok {
		return fs.ErrNotExist
	}
	delete(m.files, name)
	return nil
}

type mapFile struct {
	name string
	data []byte
	pos  int
}

func (f *mapFile) Close() error {
	return nil
}

func (f *mapFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: f.name, size: int64(len(f.data))}, nil
}

func (f *mapFile) Read(b []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *mapFile) Write(b []byte) (int, error) {
	if f == nil {
		return 0, errors.New("file is nil")
	}
	f.data = append(f.data, b...)
	return len(b), nil
}

type fileInfo struct {
	name string
	size int64
}

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Size() int64        { return fi.size }
func (fi fileInfo) Mode() fs.FileMode  { return 0o444 }
func (fi fileInfo) ModTime() time.Time { return time.Time{} }
func (fi fileInfo) IsDir() bool        { return false }
func (fi fileInfo) Sys() interface{}   { return nil }
