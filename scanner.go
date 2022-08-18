package filecp

import "fmt"

func NewScanner(ip, path string) Scanner {
	if ip != "" {
		return new_remote_scanner(ip, path)
	}
	return new_local_scanner(ip, path)
}

func NewOperator(ip, path string) Operator {
	if ip != "" {
		return new_remote_operater(ip, path)
	}
	return new_local_operater(ip, path)
}

func (c *ScanInfos) cp(t *FileCp, fip, fpath, tip, tpath string) error {
	r := NewOperator(fip, fpath)
	w := NewOperator(tip, tpath)
	err := t.copy_excute(r, w)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (c *ScanInfos) CopyAll(t *FileCp, tip, tpath string) error {
	if c.files == nil {
		return c.cp(t, c.ip, c.path, tip, tpath)
	} else {
		for _, d := range c.files {
			from := c.path + "/" + d
			to := tpath + "/" + d
			err := c.cp(t, c.ip, from, tip, to)
			if err != nil {
				return err
			}
		}
	}
	if c.dirs == nil {
		return nil
	}
	for _, d := range c.dirs {
		from := c.path + "/" + d
		to := tpath + "/" + d
		op_to := NewOperator(tip, to)
		err := op_to.CreateDir()
		if err != nil {
			fmt.Println(err)
			return err
		}
		scan_from := NewScanner(c.ip, from)
		fs, err := scan_from.Scan()
		if err != nil {
			fmt.Println(err)
			return err
		}
		err = fs.CopyAll(t, tip, to)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}
