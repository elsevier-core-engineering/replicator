package timestamp

import (
	"fmt"
	"strconv"
	"time"
)

//To deal with Unix timestamps
type Timestamp time.Time

func (t *Timestamp) MarshalJSON() ([]byte, error) {
	ts := time.Time(*t).Unix()
	stamp := fmt.Sprint(ts)
	return []byte(stamp), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	*t = Timestamp(time.Unix(int64(ts/1000), 0))
	return nil
}

func (t *Timestamp) String() string {
	return time.Time(*t).String()
}
