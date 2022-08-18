package filecp

import "strings"

type RemoteScanner struct {
	BasicScanner
}

type RemoteOperator struct {
	BasicOperator
}

func parse_remote_path(path string) (ip, file string) {
	s := strings.Split(path, ":")
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
