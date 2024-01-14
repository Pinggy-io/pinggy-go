package pinggy_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

type mapFS struct {
	files map[string][]byte
}

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
	log.Println("Request for: ", name)
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

func TestConnection(t *testing.T) {
	l, err := pinggy.Connect(pinggy.HTTP)
	if err != nil {
		t.Fatalf("Test failed: %v\n", err)
	}
	fmt.Println(l.Addr())
	l.Close()
}

func TestFileServing(t *testing.T) {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	fname := "hello"
	fdata := []byte("This is data")

	var fs fs.FS = NewMapFS(map[string][]byte{fname: fdata})
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{Server: "a.pinggy.io"})
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	urls := pl.RemoteUrls()
	pl.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println("Connected, ", urls)
	// fs = os.DirFS("/tmp/")
	go func() { fmt.Println("Error: ", pl.ServeHttp(fs)) }()
	for _, url := range urls {
		url += "/" + fname
		response, err := http.Get(url)
		if err != nil {
			log.Println("Error:", err)
		} else {
			if response.StatusCode != 200 {
				log.Println("Status mismatch: ", response.StatusCode, " "+url)
			} else {
				fmt.Println("Content-Length: ", response.Header.Get("Content-length"))
				body, _ := ioutil.ReadAll(response.Body)
				if string(body) != string(fdata) {
					fmt.Println("Not matching")
				} else {
					fmt.Println("Matching for: ", url)
				}
			}
		}
	}
	// time.Sleep(time.Second * 20)
	pl.Close()
}
