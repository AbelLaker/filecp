package filecp

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

type RemoteScanner struct {
	BasicScanner
}

type RemoteOperator struct {
	BasicOperator
	conn net.Conn
	EOF  bool
}

func parse_remote_path(path string) (ip, file string) {
	s := strings.SplitN(path, ":", 2)
	if len(s) == 2 {
		return s[0], s[1]
	}
	return "", path
}

func new_remote_scanner(ip, path string) *RemoteScanner {
	scan := &RemoteScanner{}
	scan.ip = ip
	scan.path = path
	return scan
}

func new_remote_operater(ip, path string) *RemoteOperator {
	//var err error
	p := &RemoteOperator{}
	p.ip = ip
	p.path = path
	return p
}

/////////////////////////////////
func (d *RemoteScanner) Scan() (*ScanInfos, error) {
	op := RemoteOperator{}
	op.ip = d.ip
	op.path = d.path
	defer op.Close()
	sub := SubCmdStatFile{Path: op.path}
	ds, _ := json.Marshal(sub)
	r := op.remote_cmd_excute(TOP_CMD_SCANDIR, string(ds))
	if r.Code != 0 {
		return nil, errors.New(r.Des)
	}
	fs := &ScanInfos{}
	json.Unmarshal([]byte(r.Ext), fs)
	fs.Ip = d.ip
	fs.Path = d.path
	return fs, nil
}

////////////////////////////////

func (p *RemoteOperator) Open(mode int) error {
	p.mode = mode
	sub := &SubCmdOpenFile{}
	sub.Mode = mode
	sub.Path = p.path
	ds, _ := json.Marshal(sub)
	r := p.remote_cmd_excute(TOP_CMD_OPENFILE, string(ds))
	if r.Code != 0 {
		return errors.New(r.Des)
	}
	fmt.Println("Open", p.ip, ":", p.path, r.Ext)
	return nil
}

func (p *RemoteOperator) Stat() (*FileInfo, error) {
	sub := &SubCmdStatFile{}
	sub.Path = p.path
	ds, _ := json.Marshal(sub)
	r := p.remote_cmd_excute(TOP_CMD_STATFILE, string(ds))
	if r.Code != 0 {
		return nil, errors.New(r.Des)
	}
	st := &FileInfo{}
	err := json.Unmarshal([]byte(r.Ext), st)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (p *RemoteOperator) Seek(n int64) (int64, error) {
	p.EOF = false
	if p.conn == nil {
		return 0, errors.New("p.conn is nil")
	}
	b := make([]byte, TCP_CMD_HEAD_LEN)
	format_cmd_head(b, TCP_CMD_HEAD_LEN, TCP_CMD_SEEK_DATA, 0)
	format_cmd_head_extern_int64(b, n)
	_, err := p.conn.Write(b[:TCP_CMD_HEAD_LEN])
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (p *RemoteOperator) Read(b []byte) (int, error) {
	if p.EOF {
		return 0, io.EOF
	}
	d := make([]byte, TCP_CMD_HEAD_LEN)
	format_cmd_head(d, TCP_CMD_HEAD_LEN, TCP_CMD_READ_DATA, 0)
	format_cmd_head_extern_int32(d, int32(len(b)))
	_, err := p.conn.Write(d)
	if err != nil {
		return 0, err
	}
	return p.remote_data_read(d, b)
}

func (p *RemoteOperator) Write(b []byte) (int, error) {
	d := make([]byte, TCP_CMD_HEAD_LEN)
	format_cmd_head(d, TCP_CMD_HEAD_LEN+len(b), TCP_CMD_WRITE_DATA, 0)
	p.conn.Write(d)
	return p.conn.Write(b)
}

func (p *RemoteOperator) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *RemoteOperator) CreateDir() error {
	sub := &SubCmdStatFile{}
	sub.Path = p.path
	ds, _ := json.Marshal(sub)
	r := p.remote_cmd_excute(TOP_CMD_MKDIR, string(ds))
	if r.Code != 0 {
		return errors.New(r.Des)
	}
	return nil
}

func (p *RemoteOperator) IsDir() bool {
	d, _ := p.Stat()
	if d == nil {
		return false
	}
	return d.IsDir
}

////////////////////
func (p *RemoteOperator) remote_data_read(d, b []byte) (int, error) {
	off := 0
	for {
		n, err := p.conn.Read(d[off:TCP_CMD_HEAD_LEN])
		if err != nil {
			return 0, err
		}
		off += n
		if off >= TCP_CMD_HEAD_LEN {
			break
		}
	}
	if d[0] != TCP_CMD_HEAD_TAG0 || d[1] == TCP_CMD_HEAD_TAG1 {
		return 0, errors.New("Not Format Cmd Head")
	}
	if d[TCP_CMD_TYPE_IDX] != TCP_CMD_DATA {
		return 0, errors.New("Not Read Cmd Data")
	}
	l := int(binary.LittleEndian.Uint32(d[2:]))
	if l > len(b) {
		return 0, errors.New("Read Data too long")
	}
	l = l - off
	if l > 0 {
		off = 0
		for {
			n, err := p.conn.Read(b[off:])
			if err != nil {
				return 0, err
			}
			off += n
			if off >= l {
				break
			}
		}
	}

	if d[TCP_CMD_STATE_IDX] == TCP_CMD_STATE_ERR {
		r := &CmdResult{}
		json.Unmarshal(b[:l], r)
		return 0, errors.New(r.Des)
	}
	if l < len(b) {
		p.EOF = true
	}
	return l, nil
}

func (p *RemoteOperator) remote_cmd_excute(cmd, sub string) *CmdResult {
	var err error
	r := &CmdResult{}
	if p.conn == nil {
		p.conn, err = connect_remote(p.ip + DEFAULT_FILE_SEVER_PORT)
		if err != nil {
			r.Code = -1
			r.Des = err.Error()
			return r
		}
	}

	b := make([]byte, 32*1024)
	c := &TopCmd{}
	c.Cmd = cmd
	c.SubCmd = sub
	d, _ := json.Marshal(c)
	format_cmd_head(b, len(d)+TCP_CMD_HEAD_LEN, TCP_CMD_EXCUTE, 0)
	p.conn.Write(b[:TCP_CMD_HEAD_LEN])
	_, err = p.conn.Write(d)
	if err != nil {
		r.Code = -1
		r.Des = err.Error()
		return r
	}

	dd, n, err := read_tcp_cmd(p.conn, b)
	if err != nil {
		r.Code = -1
		r.Des = err.Error()
		return r
	}

	err = json.Unmarshal(dd[TCP_CMD_HEAD_LEN:n], r)
	if err != nil {
		r.Code = -1
		r.Des = err.Error()
		return r
	}
	return r
}
