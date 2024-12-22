package humanreadable

import (
	"fmt"
	"time"
)

type Second time.Duration

func (sec Second) String() string {
	var s, m, h, d string

	sec = Second(time.Duration(sec) / time.Second)

	tm := sec / 60
	ts := sec % 60

	if ts > 0 {
		s = fmt.Sprintf("%ds", ts)
	}

	if tm > 0 {
		th := tm / 60
		tm := tm % 60

		if tm > 0 {
			m = fmt.Sprintf("%dm", tm)
		}

		if th > 0 {
			td := th / 24
			th := th % 24

			if th > 0 {
				h = fmt.Sprintf("%dh", th)
			}

			if td > 0 {
				d = fmt.Sprintf("%dd", td)
			}
		}
	}

	return d + h + m + s
}
