package logs

type Log struct {
	Filename string `json:"filename,omitempty"`
	Date     int    `json:"date,omitempty"`
	Size     int    `json:"size,omitempty"`
}
