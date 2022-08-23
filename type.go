package filecp

import (
	"errors"
	"hash"
	"sync"
)

const (
	NO_LIMIT_SPEEK = -1

	CP_STATE_RUNNING = 1
	CP_STATE_STOP    = 2
	//CP_STATE_ERROR   = 4

	FILe_OPEN_MODE_READ  = 1
	FILe_OPEN_MODE_WRITE = 2
	FILe_COPY_WITH_MD5   = 4
	FILe_MODE_LOCAL      = 8

	DEFAULT_FILE_SEVER_URL  = "0.0.0.0:8864"
	DEFAULT_FILE_SEVER_PORT = ":8864"
	TEST_ID_STRING          = "test1"
	TCP_CMD_HEAD_LEN        = 16
	TCP_CMD_HEAD_TAG0       = 0xff
	TCP_CMD_HEAD_TAG1       = 0xfe
	TCP_CMD_TYPE_IDX        = 6
	TCP_CMD_STATE_IDX       = 7
	TCP_CMD_EXTERN          = 8

	TCP_CMD_LOGIN      = 'l'
	TCP_CMD_WRITE_DATA = 'w'
	TCP_CMD_READ_DATA  = 'r'
	TCP_CMD_SEEK_DATA  = 's'
	TCP_CMD_EXCUTE     = 'e'
	TCP_CMD_DATA       = 'd'

	TCP_CMD_STATE_ERR = 'e'

	TOP_CMD_OPENFILE = "openfile"
	TOP_CMD_STATFILE = "stat"
	TOP_CMD_MKDIR    = "mkdir"
	TOP_CMD_FINISH   = "finish"
	TOP_CMD_SCANDIR  = "scan"
	TOP_CMD_MD5SIZE  = "md5size"
	TOP_CMD_MD5STR   = "md5str"
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
	Ip    string
	Path  string
	Files []string
	Dirs  []string
}

type FileInfo struct {
	Size  int64
	IsDir bool
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
	Stat() (*FileInfo, error)
	Seek(n int64) (int64, error)
	Read([]byte) (n int, err error)
	Write([]byte) (n int, err error)
	Close()
	CreateDir() error
	Path() string
	Finish() error
	GetMd5RecSize() (int64, error)
	GetMd5String() (string, error)
}

type BasicOperator struct {
	ip      string
	path    string
	mode    int
	bmd5    bool
	bfinish bool
	md5     hash.Hash
	md5size int64
}

func (p *BasicOperator) Open(mode int) error {
	return errors.New("Err: unkown Operator Open!")
}

func (p *BasicOperator) Stat() (*FileInfo, error) {
	return nil, nil
}

func (p *BasicOperator) Seek(n int64) (int64, error) {
	return n, nil
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

func (p *BasicOperator) Finish() error {
	p.bfinish = true
	return nil
}

func (p *BasicOperator) GetMd5RecSize() (int64, error) {
	return 0, nil
}

func (p *BasicOperator) GetMd5String() (string, error) {
	return "", nil
}
