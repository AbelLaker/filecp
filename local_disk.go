package filecp

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AbelLaker/md5"
	"hash"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type LocalScanner struct {
	BasicScanner
}

type LocalOperator struct {
	BasicOperator
	f      *os.File
	offset int64
}

type file_md5_cache struct {
	Md5     hash.Hash
	Md5size int64
}

type file_dig_cache struct {
	Md5     md5.Digest
	Md5size int64
}

var cache_lock sync.Mutex
var cache_md5 map[string]file_md5_cache

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
		Ip:   d.ip,
		Path: d.path,
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
	fs.Files = make([]string, 0)
	fs.Dirs = make([]string, 0)
	for _, fi := range dir {
		if fi.IsDir() {
			fs.Dirs = append(fs.Dirs, fi.Name())
		} else {
			fs.Files = append(fs.Files, fi.Name())
		}
	}
	//fmt.Println("LocalScanner:Scan", fs)
	return fs, nil
}

////////////////////////////////////

func (p *LocalOperator) Open(mode int) error {
	var err error
	p.mode = mode
	p.mode |= FILe_MODE_LOCAL
	if (mode & FILe_COPY_WITH_MD5) > 0 {
		p.bmd5 = true
		if p.md5 == nil {
			cache := FetchMd5Cache(p.path)
			if cache.Md5 == nil {
				cache = GetMd5FromFile(p.path)
			}
			if cache.Md5 == nil {
				cache.Md5 = md5.New()
				cache.Md5size = 0
			}
			p.md5 = cache.Md5
			p.md5size = cache.Md5size
		}
	}
	if (mode & FILe_OPEN_MODE_WRITE) > 0 {
		p.f, err = os.OpenFile(p.path, os.O_CREATE|os.O_RDWR, 0644) //|os.O_APPEND
	} else {
		p.f, err = os.Open(p.path)
	}
	return err
}

func (p *LocalOperator) Stat() (*FileInfo, error) {
	st := &FileInfo{}
	s, err := os.Stat(p.path)
	if err == nil {
		st.Size = s.Size()
		st.IsDir = s.IsDir()
		return st, err
	} else {
		return nil, err
	}
}

func (p *LocalOperator) Seek(n int64) (int64, error) {
	if p.f == nil {
		return 0, errors.New("p.f is nil")
	}
	p.offset = n
	return p.f.Seek(n, os.SEEK_SET)
}

func (p *LocalOperator) Read(b []byte) (int, error) {
	if p.f == nil {
		return 0, errors.New("p.f is nil")
	}
	n, err := p.f.Read(b)
	if p.bmd5 && n > 0 {
		p.offset += int64(n)
		if p.offset > p.md5size {
			off := int64(n) - (p.offset - p.md5size)
			if off < 0 {
				off = 0
			}
			p.md5.Write(b[off:n])
			p.md5size = p.offset
		}
	}
	return n, err
}

func (p *LocalOperator) Write(b []byte) (int, error) {
	if p.f == nil {
		return 0, errors.New("p.f is nil")
	}
	n, err := p.f.Write(b)
	if p.bmd5 && n > 0 {
		p.offset += int64(n)
		if p.offset > p.md5size {
			off := int64(n) - (p.offset - p.md5size)
			if off < 0 {
				off = 0
			}
			p.md5.Write(b[off:n])
			p.md5size = p.offset
		}
	}
	return n, err
}

func (p *LocalOperator) Close() {
	fmt.Println("LocalOperator: close p.bfinish:", p.bfinish)
	if p.md5 != nil {
		if p.bfinish == false {
			cache := file_md5_cache{Md5: p.md5, Md5size: p.md5size}
			if SaveMd5Cache(p.path, cache) != nil {
				SaveMd5ToFile(&cache, p.path)
				fmt.Println(p.path, "save md5 to file, md5size:", p.md5size)
			} else {
				fmt.Println(p.path, "save md5 to cache, md5size:", p.md5size)
			}
		} else {
			RemoveFileMd5Rec(p.path)
		}
	}
	if p.f != nil {
		p.f.Close()
	}
}

func (p *LocalOperator) CreateDir() error {
	return CreateDir(p.path)
}

func (p *LocalOperator) IsDir() bool {
	d, _ := os.Stat(p.path)
	if d == nil {
		return false
	}
	return d.IsDir()
}

func (p *LocalOperator) GetMd5RecSize() (int64, error) {
	if p.md5 == nil {
		cache := FetchMd5Cache(p.path)
		if cache.Md5 == nil {
			cache = GetMd5FromFile(p.path)
		}
		if cache.Md5 == nil {
			cache.Md5 = md5.New()
			cache.Md5size = 0
		}
		p.md5 = cache.Md5
		p.md5size = cache.Md5size
	}
	return p.md5size, nil
}

func (p *LocalOperator) GetMd5String() (string, error) {
	if p.md5 != nil {
		return fmt.Sprintf("%x", p.md5.Sum(nil)), nil
	}
	return "", nil
}

/////////////////////

func FileExist(p string) bool {
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

func CreateDir(path string) error {
	if FileExist(path) == false {
		return os.MkdirAll(path, 0775)
	}
	return nil
}

func SaveMd5ToFile(cache *file_md5_cache, path string) error {
	d, _ := json.Marshal(&cache)
	err := CreateDir("/tmp/filecp/")
	fmt.Println("CreateDir", err)
	f := file_md5_rec_path(path)
	return ioutil.WriteFile(f, d, 0664)
}

func GetMd5FromFile(path string) file_md5_cache {
	cache := file_md5_cache{}
	f := file_md5_rec_path(path)
	if FileExist(f) {
		d, err := ioutil.ReadFile(f)
		r := &file_dig_cache{}
		if err == nil {
			err = json.Unmarshal(d, r)
			if err == nil {
				cache.Md5size = r.Md5size
				cache.Md5 = &r.Md5
			}
		}
	}
	return cache
}

func RemoveFileMd5Rec(path string) error {
	f := file_md5_rec_path(path)
	if FileExist(f) {
		return os.Remove(f)
	}
	return nil
}

func file_md5_rec_path(path string) string {
	h := md5.New()
	h.Write([]byte(path))
	return "/tmp/filecp/" + fmt.Sprintf("%x", h.Sum(nil))
}

func InitCache() {
	cache_md5 = make(map[string]file_md5_cache, 0)
	dir, err := ioutil.ReadDir("/tmp/filecp/")
	if err != nil {
		return
	}
	for _, fi := range dir {
		if fi.IsDir() == false {
			f := "/tmp/filecp/" + fi.Name()
			if time.Now().Sub(fi.ModTime()).Hours() > 7*24 {
				os.Remove(f)
				continue
			}
			cache := file_md5_cache{}
			d, err := ioutil.ReadFile(f)
			if err == nil {
				r := &file_dig_cache{}
				err = json.Unmarshal(d, r)
				if err == nil {
					cache.Md5size = r.Md5size
					cache.Md5 = &r.Md5
					cache_md5[f] = cache
				}
			}
		}
	}
}

func FetchMd5Cache(path string) file_md5_cache {
	f := file_md5_rec_path(path)
	cache_lock.Lock()
	defer cache_lock.Unlock()
	cache := cache_md5[f]
	if cache.Md5 != nil {
		delete(cache_md5, f)
	}
	return cache
}

func SaveMd5Cache(path string, cache file_md5_cache) error {
	if cache_md5 == nil {
		return errors.New("no cache for save")
	}
	f := file_md5_rec_path(path)
	cache_lock.Lock()
	defer cache_lock.Unlock()
	cache_md5[f] = cache
	return nil
}

func DumpCacheToFile() {
	if cache_md5 == nil {
		return
	}
	cache_lock.Lock()
	defer cache_lock.Unlock()
	CreateDir("/tmp/filecp/")
	for f, cache := range cache_md5 {
		d, _ := json.Marshal(&cache)
		ioutil.WriteFile(f, d, 0664)
	}
}
