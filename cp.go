package filecp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

func NewCpTask(from, to string) *FileCp {
	t := &FileCp{}
	t.from = from
	t.to = to
	t.max_speek = NO_LIMIT_SPEEK
	return t
}

///////////////////////////////////////////

func (t *FileCp) GetPath() (from string, to string) {
	return t.from, t.to
}

func (t *FileCp) Stop() {
	t.set_state(CP_STATE_STOP)
}

func (t *FileCp) SetCheckMd5(b bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.check_md5 = b
}

func (t *FileCp) IsCheckMd5() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.check_md5
}

func (t *FileCp) GetMaxSpeek() int64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.max_speek
}

func (t *FileCp) SetMaxSpeek(m int64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.max_speek = m
}

func (t *FileCp) Stat(name string) (info os.FileInfo, err error) {
	return os.Stat(name)
}

func (t *FileCp) set_state_bit(s int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state |= s
}

func (t *FileCp) set_state(s int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state = s
}

func (t *FileCp) clear_state_bit(s int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state &= ^s
}

func (t *FileCp) get_state() int {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.state
}

func (t *FileCp) is_stop() bool {
	if (t.get_state() & CP_STATE_STOP) > 0 {
		return true
	}
	return false
}

func (t *FileCp) Copy() error {
	state := t.get_state()
	if (state & CP_STATE_RUNNING) > 0 {
		fmt.Println("Copy from", t.from, "to", t.to, "is running")
		return nil
	}
	t.set_state(CP_STATE_RUNNING)
	defer t.clear_state_bit(CP_STATE_RUNNING)
	fmt.Println("Start Copy from", t.from, "to", t.to)
	fip, fpath := parse_remote_path(t.from)
	tip, tpath := parse_remote_path(t.to)
	scan_from := NewScanner(fip, fpath)
	fs, err := scan_from.Scan()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return fs.CopyAll(t, tip, tpath)

}

func (t *FileCp) info_copy(r Operator, w Operator) error {
	st_f, err := r.Stat()
	if err != nil {
		return err
	}
	t.r_size = st_f.Size
	st_t, err := w.Stat()
	if err != nil {
		t.w_size = 0
	} else {
		t.w_size = st_t.Size
	}
	if t.w_size == t.r_size {
		return nil
	}
	if t.check_md5 {
		wn, _ := w.GetMd5RecSize()
		rn, _ := r.GetMd5RecSize()
		if wn < t.w_size {
			t.w_size = wn
		}
		if rn < t.w_size {
			t.w_size = rn
		}
	}

	return nil
}

func (t *FileCp) seek_copy(r Operator, w Operator) error {
	n, err := r.Seek(t.w_size)
	if err != nil {
		return err
	}
	if n != t.w_size {
		t.w_size = n
	}

	n, err = w.Seek(t.w_size)
	if err != nil {
		return err
	}
	if n != t.w_size {
		t.w_size = n
		return t.seek_copy(r, w)
	}
	return nil
}
func (t *FileCp) copy_finish(r Operator, w Operator) error {
	r.Finish()
	w.Finish()
	if t.check_md5 {
		rmd5, err := r.GetMd5String()
		if err != nil {
			return err
		}
		wmd5, err := w.GetMd5String()
		if err != nil {
			return err
		}
		if rmd5 != wmd5 {
			return errors.New("md5 not eq: " + rmd5 + " != " + wmd5)
		}
		fmt.Println(r.Path(), "md5 eq:", rmd5)
	}
	fmt.Println(r.Path(), "copy finish")
	return nil
}

func (t *FileCp) copy_loop(r Operator, w Operator) error {
	buf := make([]byte, 1024*32)
	t.cp_bytes = 0
	for {
		if t.is_stop() {
			return nil
		}
		maxspeed := t.GetMaxSpeek()
		if maxspeed == 0 {
			time.Sleep(time.Second)
			continue
		}
		tn := time.Now()
		subt := 0
		for {
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					return t.copy_finish(r, w)
				}
				return err
			}
			_, err = w.Write(buf[:n])
			if err != nil {
				return err
			}
			t.w_size += int64(n)
			subt += n
			t.cp_bytes += int64(n)
			if t.w_size >= t.r_size {
				return t.copy_finish(r, w)
			}
			if int64(subt) >= maxspeed {
				break
			}
		}
		if maxspeed != NO_LIMIT_SPEEK {
			time.Sleep(time.Second - time.Now().Sub(tn))
		}
	}
	return nil
}

func (t *FileCp) copy_excute(r Operator, w Operator) error {
	//fmt.Println("copy_excute:", r, w)
	err := t.info_copy(r, w)
	if err != nil {
		return err
	}
	if t.r_size == t.w_size {
		fmt.Println(w.Path(), "has finished!")
		return nil
	}
	rmod := FILe_OPEN_MODE_READ
	wmod := FILe_OPEN_MODE_WRITE
	if t.IsCheckMd5() {
		rmod |= FILe_COPY_WITH_MD5
		wmod |= FILe_COPY_WITH_MD5
	}
	err = r.Open(rmod)
	if err != nil {
		return err
	}
	defer r.Close()
	err = w.Open(wmod)
	if err != nil {
		return err
	}
	defer w.Close()

	err = t.seek_copy(r, w)
	if err != nil {
		return err
	}

	err = t.copy_loop(r, w)
	return err
}

/////////////////////////////////////////////
