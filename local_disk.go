package filecp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type LocalScanner struct {
	BasicScanner
}

type LocalOperator struct {
	BasicOperator
	f *os.File
}

func new_local_scanner(ip, path string) *LocalScanner {
	scan := &LocalScanner{}
	scan.ip = ip
	scan.path = path
	return scan
}

func new_local_operater(ip, path string) *LocalOperator {
	p := &LocalOperator{}
	p.ip = ip
	p.path = path
	return p
}

////////////////////////////////////////////////////

func (d *LocalScanner) Scan() (*ScanInfos, error) {
	fs := &ScanInfos{
		ip:   d.ip,
		path: d.path,
	}
	f, err := os.Stat(d.path)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errors.New("Err: os.Stat is nil!")
	}
	if !f.IsDir() {
		return fs, nil
	}
	fmt.Println("scanning", d.path)
	dir, err := ioutil.ReadDir(d.path)
	if err != nil {
		return nil, err
	}
	fs.files = make([]string, 0)
	fs.dirs = make([]string, 0)
	for _, fi := range dir {
		if fi.IsDir() {
			fs.dirs = append(fs.dirs, fi.Name())
		} else {
			fs.files = append(fs.files, fi.Name())
		}
	}
	//fmt.Println("LocalScanner:Scan", fs)
	return fs, nil
}

////////////////////////////////////

func (p *LocalOperator) Open(mode int) error {
	var err error
	p.mode = mode
	if mode == FILe_OPEN_MODE_WRITE {
		p.f, err = os.OpenFile(p.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	} else {
		p.f, err = os.Open(p.path)
	}
	return err
}

func (p *LocalOperator) Stat() (os.FileInfo, error) {
	return os.Stat(p.path)
}

func (p *LocalOperator) Seek(n int64) error {
	_, err := p.f.Seek(n, os.SEEK_SET)
	return err
}

func (p *LocalOperator) Read(b []byte) (n int, err error) {
	return p.f.Read(b)
}

func (p *LocalOperator) Write(b []byte) (n int, err error) {
	return p.f.Write(b)
}

func (p *LocalOperator) Close() {
	p.f.Close()
}

func (p *LocalOperator) CreateDir() error {
	return os.MkdirAll(p.path, 0775)
}

func (p *LocalOperator) IsDir() bool {
	d, _ := os.Stat(p.path)
	if d == nil {
		return false
	}
	return d.IsDir()
}
