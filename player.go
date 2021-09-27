package sn

type PID int

const NoPID PID = 0

func (pid PID) ToIndex() int {
	return int(pid) - 1
}

func ToPID(i int) PID {
	return PID(i + 1)
}

const found = true

func (h Header) IndexFor(uid int64) (int, bool) {
	for i, uid2 := range h.UserIDS {
		if uid == uid2 {
			return i, found
		}
	}
	return -1, !found
}
