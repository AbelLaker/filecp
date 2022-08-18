package filecp

import (
	"errors"
	"os"
	"sync"
)

const (
	NO_LIMIT_SPEEK = -1

	CP_STATE_RUNNING = 1
	CP_STATE_STOP    = 2
	//CP_STATE_ERROR   = 4

	FILe_OPEN_MODE_READ  = 1
	FILe_OPEN_MODE_WRITE = 2
)

type FileCp struct {
	from      string
	to        string
	max_speek int64
	state     int
	check_md5 bool
	lock      sync.Mutex
	r_size    int64
	w_size    int64
	cp_bytes  int64
}

type ScanInfos struct {
	ip    string
	path  string
	files []string
	dirs  []string
}

////////////////////////////////
type Scanner interface {
	Scan() (*ScanInfos, error)
}

type BasicScanner struct {
	ip   string
	path string
}

func (s *BasicScanner) Scan() (*ScanInfos, error) {
	return nil, errors.New("Err: unkown scanner!")
}

type Operator interface {
	Open(mode int) error
	Stat() (os.FileInfo, error)
	Seek(n int64) error
	Read([]byte) (n int, err error)
	Write([]byte) (n int, err error)
	Close()
	CreateDir() error
	Path() string
}

type BasicOperator struct {
	ip   string
	path string
	mode int
}

func (p *BasicOperator) Open(mode int) error {
	return errors.New("Err: unkown Operator Open!")
}

func (p *BasicOperator) Stat() (os.FileInfo, error) {
	return nil, nil
}

func (p *BasicOperator) Seek(n int64) error {
	return nil
}

func (p *BasicOperator) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (p *BasicOperator) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (p *BasicOperator) Close() {}

func (p *BasicOperator) CreateDir() error {
	return nil
}

func (p *BasicOperator) Path() string {
	return p.path
}
