package sn

type PID int

const NoPID PID = 0

func (pid PID) ToIndex() int {
	return int(pid) - 1
}

func ToPID(i int) PID {
	return PID(i + 1)
}
