package filecp

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
)

type file_server_operator struct {
	conn net.Conn
	op   Operator
}

type CmdResult struct {
	Code int
	Des  string
	Ext  string
}

type TopCmd struct {
	Cmd    string
	SubCmd string
}

type SubCmdOpenFile struct {
	Mode int
	Path string
}

type SubCmdStatFile struct {
	Path string
}

type Md5Info struct {
	Md5Size int64
	Md5str  string
}

type CmdConnect struct {
	ID string
}

//url : 0.0.0.0:8864
func FileServerRun(url string) error {
	if url == "" {
		url = DEFAULT_FILE_SEVER_URL
	}
	fmt.Println("listen:", url)
	listenner, err := net.Listen("tcp", url)
	if err != nil {
		return err
	}
	defer listenner.Close()
	InitCache()
	for {
		conn, err1 := listenner.Accept()
		if err1 != nil {
			fmt.Println("ERR:", err1)
			continue
		}
		go func() {
			defer conn.Close()
			b := make([]byte, 48*1024)
			s := file_server_operator{
				conn: conn,
			}
			{
				d, n, err := read_tcp_cmd(s.conn, b)
				if err != nil {
					return
				}
				if !s.is_legal_login(d[:n]) {
					return
				}
			}
			s.operate_cmd(b)
		}()
	}
	return nil
}

func (s *file_server_operator) operate_cmd(b []byte) error {
	for {
		{
			d, n, err := read_tcp_cmd(s.conn, b)
			if err != nil {
				if s.op != nil {
					s.op.Close()
				}
				//fmt.Println(err)
				return err
			}
			switch d[TCP_CMD_TYPE_IDX] {
			case TCP_CMD_EXCUTE:
				s.excute(d[TCP_CMD_HEAD_LEN:n])
				break
			case TCP_CMD_WRITE_DATA:
				_, err := s.Write(d[TCP_CMD_HEAD_LEN:n])
				if err != nil {
					return err
				}
				break
			case TCP_CMD_READ_DATA:
				_, err := s.Read(d)
				if err != nil {
					return err
				}
				break
			case TCP_CMD_SEEK_DATA:
				_, err := s.Seek(d)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (s *file_server_operator) Seek(b []byte) (int64, error) {
	l := binary.LittleEndian.Uint64(b[TCP_CMD_EXTERN:])
	if s.op != nil {
		return s.op.Seek(int64(l))
	} else {
		return 0, errors.New("Not Open File")
	}
}

func (s *file_server_operator) Write(b []byte) (int, error) {
	if s.op != nil {
		return s.op.Write(b)
	} else {
		return 0, errors.New("Not Open File")
	}
}

func (s *file_server_operator) Read(b []byte) (int, error) {
	r := CmdResult{}
	r.Code = 0
	l := int(binary.LittleEndian.Uint32(b[TCP_CMD_EXTERN:]))
	if l > len(b) || l <= 0 {
		r.Code = -1
		r.Des = "Read len err"
	} else if s.op != nil {
		n, err := s.op.Read(b[TCP_CMD_HEAD_LEN:(TCP_CMD_HEAD_LEN + l)])
		if err != err {
			if err != io.EOF {
				r.Code = -1
				r.Des = err.Error()
			}
		}
		l = TCP_CMD_HEAD_LEN + n
	} else {
		r.Code = -1
		r.Des = "Not Open File"
	}
	if r.Code != 0 {
		d, _ := json.Marshal(&r)
		format_cmd_head(b, TCP_CMD_HEAD_LEN+len(d), TCP_CMD_DATA, TCP_CMD_STATE_ERR)
		s.conn.Write(b[:TCP_CMD_HEAD_LEN])
		s.conn.Write(d)
		return 0, errors.New(r.Des)
	} else {
		format_cmd_head(b, l, TCP_CMD_DATA, 0)
		return s.conn.Write(b[:l])
	}
}

func (s *file_server_operator) openfile(sub string) (string, error) {
	c := &SubCmdOpenFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	if s.op == nil {
		ip, path := parse_remote_path(c.Path)
		s.op = NewOperator(ip, path)
	}
	err = s.op.Open(c.Mode)
	if err != nil {
		return "", err
	}
	fmt.Println("Open", s.op.Path(), "Success")
	return "Success", nil
}

func (s *file_server_operator) statfile(sub string) (string, error) {
	c := &SubCmdStatFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	if s.op == nil {
		ip, path := parse_remote_path(c.Path)
		s.op = NewOperator(ip, path)
	}
	st, err := s.op.Stat()
	if err != nil {
		return "", err
	}
	d, _ := json.Marshal(st)
	return string(d), nil
}

func (s *file_server_operator) mkdir(sub string) (string, error) {
	c := &SubCmdStatFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	if s.op == nil {
		ip, path := parse_remote_path(c.Path)
		s.op = NewOperator(ip, path)
	}
	err = s.op.CreateDir()
	if err != nil {
		return "", err
	}
	return "", nil
}

func (s *file_server_operator) scan(sub string) (string, error) {
	c := &SubCmdStatFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	ip, path := parse_remote_path(c.Path)
	scan := NewScanner(ip, path)
	fs, err := scan.Scan()
	if err != nil {
		return "", err
	}
	d, _ := json.Marshal(fs)
	return string(d), nil
}

func (s *file_server_operator) md5size(sub string) (string, error) {
	c := &SubCmdStatFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	if s.op == nil {
		ip, path := parse_remote_path(c.Path)
		s.op = NewOperator(ip, path)
	}
	md5size, err := s.op.GetMd5RecSize()
	if err != nil {
		return "", err
	}
	info := &Md5Info{Md5Size: md5size, Md5str: ""}
	d, _ := json.Marshal(info)
	fmt.Println("md5size:", c.Path, "size:", info.Md5Size)
	return string(d), nil
}

func (s *file_server_operator) md5str(sub string) (string, error) {
	c := &SubCmdStatFile{}
	err := json.Unmarshal([]byte(sub), c)
	if err != nil {
		return "", err
	}
	if s.op == nil {
		ip, path := parse_remote_path(c.Path)
		s.op = NewOperator(ip, path)
	}
	md5str, err := s.op.GetMd5String()
	if err != nil {
		return "", err
	}
	info := &Md5Info{Md5Size: 0, Md5str: md5str}
	d, _ := json.Marshal(info)
	fmt.Println("md5str:", c.Path, "str:", info.Md5str)
	return string(d), nil
}

func (s *file_server_operator) excute_subcmd(cmd, sub string) (string, error) {
	if cmd == TOP_CMD_OPENFILE {
		return s.openfile(sub)
	} else if cmd == TOP_CMD_STATFILE {
		return s.statfile(sub)
	} else if cmd == TOP_CMD_MKDIR {
		return s.mkdir(sub)
	} else if cmd == TOP_CMD_SCANDIR {
		return s.scan(sub)
	} else if cmd == TOP_CMD_MD5SIZE {
		return s.md5size(sub)
	} else if cmd == TOP_CMD_MD5STR {
		return s.md5str(sub)
	} else if cmd == TOP_CMD_FINISH {
		if s.op != nil {
			s.op.Finish()
		}
		return "", nil
	}
	return "", errors.New("unkown cmd")
}

func (s *file_server_operator) excute(b []byte) error {
	c := &TopCmd{}
	r := &CmdResult{}
	r.Code = 0
	err := json.Unmarshal(b, c)
	if err == nil {
		r.Ext, err = s.excute_subcmd(c.Cmd, c.SubCmd)
	}
	if err != nil {
		r.Code = -1
		r.Des = err.Error()
		fmt.Println(r.Des)
	}
	d, _ := json.Marshal(r)
	if r.Code != 0 {
		format_cmd_head(b, TCP_CMD_HEAD_LEN+len(d), TCP_CMD_DATA, TCP_CMD_STATE_ERR)
		s.conn.Write(b[:TCP_CMD_HEAD_LEN])
		s.conn.Write(d)
		return errors.New(r.Des)
	}
	format_cmd_head(b, TCP_CMD_HEAD_LEN+len(d), TCP_CMD_DATA, TCP_CMD_STATE_ERR)
	s.conn.Write(b[:TCP_CMD_HEAD_LEN])
	_, err = s.conn.Write(d)
	return err
}

func (s *file_server_operator) is_legal_login(b []byte) bool {
	c := &CmdConnect{}
	if b[TCP_CMD_TYPE_IDX] != TCP_CMD_LOGIN {
		fmt.Println("b[TCP_CMD_TYPE_IDX] != TCP_CMD_LOGIN")
		return false
	}
	err := json.Unmarshal(b[TCP_CMD_HEAD_LEN:], c)
	if c.ID != get_id_string() || err != nil {
		fmt.Println("Err ID = ", c.ID, "-", err)
		return false
	}
	return true
}

///////////////////////////////////

func connect_remote(url string) (net.Conn, error) {
	c, err := net.Dial("tcp", url)
	if err != nil {
		fmt.Println(err)
		return c, err
	}
	b := make([]byte, TCP_CMD_HEAD_LEN)
	cmd := CmdConnect{
		ID: get_id_string(),
	}
	d, _ := json.Marshal(cmd)
	format_cmd_head(b, len(d)+TCP_CMD_HEAD_LEN, TCP_CMD_LOGIN, 0)
	c.Write(b[:TCP_CMD_HEAD_LEN])
	_, err = c.Write(d)
	if err != nil {
		c.Close()
		c = nil
	}
	return c, err
}

func format_cmd_head(b []byte, l int, t, e byte) {
	b[0] = TCP_CMD_HEAD_TAG0
	b[1] = TCP_CMD_HEAD_TAG0
	b[2] = byte(l)
	b[3] = byte(l >> 8)
	b[4] = byte(l >> 16)
	b[5] = byte(l >> 24)
	b[TCP_CMD_TYPE_IDX] = t
	b[TCP_CMD_STATE_IDX] = e
}

func format_cmd_head_extern_int64(b []byte, l int64) {
	b[TCP_CMD_EXTERN] = byte(l)
	b[TCP_CMD_EXTERN+1] = byte(l >> 8)
	b[TCP_CMD_EXTERN+2] = byte(l >> 16)
	b[TCP_CMD_EXTERN+3] = byte(l >> 24)
	b[TCP_CMD_EXTERN+4] = byte(l >> 32)
	b[TCP_CMD_EXTERN+5] = byte(l >> 40)
	b[TCP_CMD_EXTERN+6] = byte(l >> 48)
	b[TCP_CMD_EXTERN+7] = byte(l >> 56)
}

func format_cmd_head_extern_int32(b []byte, l int32) {
	b[TCP_CMD_EXTERN] = byte(l)
	b[TCP_CMD_EXTERN+1] = byte(l >> 8)
	b[TCP_CMD_EXTERN+2] = byte(l >> 16)
	b[TCP_CMD_EXTERN+3] = byte(l >> 24)
}

func get_id_string() string {
	return TEST_ID_STRING
}

func read_tcp_cmd(conn net.Conn, b []byte) ([]byte, int, error) {
	off := 0
	for {
		n, err := conn.Read(b[off:TCP_CMD_HEAD_LEN])
		if err != nil {
			fmt.Println(err)
			return b, 0, err
		}
		off += n
		if off >= TCP_CMD_HEAD_LEN {
			break
		}
	}
	if b[0] != TCP_CMD_HEAD_TAG0 || b[1] == TCP_CMD_HEAD_TAG1 {
		return b, 0, errors.New("Not Format Cmd Head")
	}
	l := binary.LittleEndian.Uint32(b[2:])
	if l > uint32(len(b)) {
		d := make([]byte, l)
		for i := 0; i < off; i++ {
			d[i] = b[i]
		}
		for {
			n, err := conn.Read(d[off:l])
			if err != nil {
				return d, 0, err
			}
			off += n
			if off >= int(l) {
				break
			}
		}
		return d, int(l), nil
	}

	for {
		n, err := conn.Read(b[off:l])
		if err != nil {
			return b, int(l), err
		}
		off += n
		if off >= int(l) {
			break
		}
	}
	return b, int(l), nil
}

/////////////////////////
