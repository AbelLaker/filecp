package filecp

import (
	"fmt"
	"github.com/AbelLaker/md5"
	"testing"
	"time"
)

//go test -v -run TestA
func TestA(c *testing.T) {
	ip, p := parse_remote_path("127.0.0.1:./test1.bak")
	fmt.Println("ip:", ip, "path:", p)
	//ip, p = parse_remote_path("./ttttt")
	//fmt.Println("ip:", ip, "path:", p)
	//ip, p = parse_remote_path("127.0.0.1:127.0.0.1:./ttttt")
	//fmt.Println("ip:", ip, "path:", p)
	/*{
		b := make([]byte, 32*1024)
		op := NewOperator(ip, p)
		fmt.Println("op.Open", op.Open(FILe_OPEN_MODE_READ|FILe_COPY_WITH_MD5))
		n, err := op.Seek(0)
		fmt.Println("op.Seek", n, err)
		_, err = op.Read(b)
		fmt.Println("op.Read:", err)
		op.Close()
		return
	}*/

	m := md5.New()
	str := fmt.Sprintf("%x", m.Sum(nil))
	fmt.Println("md5:", str)

	t := NewCpTask("./test1", "127.0.0.1:./test1.bak")
	t.SetCheckMd5(true)
	t.Copy()
}

//go test -v -run TestB
func TestB(c *testing.T) {
	FileServerRun("")
}

//go test -v -run TestC
func TestC(c *testing.T) {
	n, err := connect_remote(DEFAULT_FILE_SEVER_URL)
	if err != nil {
		return
	}
	time.Sleep(3 * time.Second)
	n.Close()
}
