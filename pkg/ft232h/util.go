package ft232h

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/ardnew/ft232h"
	"strconv"
)

func toHexStr(v uint32) string {
	b := bytes.NewBuffer(nil)
	h := hex.NewEncoder(b)

	if err := binary.Write(h, binary.BigEndian, v); err != nil || b.Len() < 5 {
		return ""
	}

	return b.String()[4:]
}

func (ft *FT232H) vidPid() (vid string, pid string) {
	vid = strconv.Itoa(int(ft.VID()))
	pid = strconv.Itoa(int(ft.PID()))

	if vids := toHexStr(ft.VID()); vids != "" {
		vid = vids
	}
	if pids := toHexStr(ft.PID()); pids != "" {
		pid = pids
	}

	return vid, pid
}

func emptyMask(mask *ft232h.Mask) bool {
	return mask == nil || (mask.Serial == "" && mask.PID == "" && mask.VID == "" && mask.Desc == "" && mask.Index == "")
}
