package ft232h

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/yunginnanet/ft232h"
	"strconv"
)

func (ft *FT232H) vidPid() (vid string, pid string) {
	vid = strconv.Itoa(int(ft.VID()))
	pid = strconv.Itoa(int(ft.PID()))

	b := bytes.NewBuffer(nil)
	h := hex.NewEncoder(b)

	if err := binary.Write(h, binary.BigEndian, ft.VID()); err == nil && len(b.String()) > 5 {
		vid = b.String()[4:]
	}

	b.Reset()

	if err := binary.Write(h, binary.BigEndian, ft.PID()); err == nil && len(b.String()) > 5 {
		pid = b.String()[4:]
	}

	return vid, pid
}

func emptyMask(mask *ft232h.Mask) bool {
	return mask == nil || (mask.Serial == "" && mask.PID == "" && mask.VID == "" && mask.Desc == "" && mask.Index == "")
}
