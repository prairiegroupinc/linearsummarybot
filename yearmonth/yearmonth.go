package yearmonth

import (
	"fmt"
	"strconv"
	"time"
)

type YM uint16 // (year-1900)<<8 | month

func FromTime(tm time.Time) YM {
	return Make(tm.Year(), int(tm.Month()))
}

func Make(year, month int) YM {
	v, err := TryMake(year, month)
	if err != nil {
		panic(err)
	}
	return v
}

func TryMake(year, month int) (YM, error) {
	y := year - 1900
	if y < 0 || y > 255 || month < 1 || month > 12 {
		return 0, fmt.Errorf("cannot represent year %d and month %d", year, month)
	}
	return YM((year-1900)<<8 | month), nil
}

func (ym YM) Components() (year int, month int) {
	year = int(ym>>8) + 1900
	month = int(ym & 0xFF)
	return
}

func (ym YM) String() string {
	year, month := ym.Components()
	return fmt.Sprintf("%04d-%02d", year, month)
}

func Parse(s string) (YM, error) {
	if len(s) != 7 || s[4] != '-' {
		return 0, fmt.Errorf("invalid format")
	}
	year, err := strconv.Atoi(s[:4])
	if err != nil {
		return 0, fmt.Errorf("invalid year")
	}
	month, err := strconv.Atoi(s[5:])
	if err != nil {
		return 0, fmt.Errorf("invalid month")
	}
	return TryMake(year, month)
}

func (ym YM) MarshalText() ([]byte, error) {
	return []byte(ym.String()), nil
}

func (ym YM) AppendText(b []byte) ([]byte, error) {
	return append(b, ym.String()...), nil
}

func (ym *YM) UnmarshalText(data []byte) error {
	v, err := Parse(string(data))
	if err != nil {
		return err
	}
	*ym = v
	return nil
}
