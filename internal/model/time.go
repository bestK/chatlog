package model

import (
	"fmt"
	"time"
)

// JSONTime 自定义时间类型，用于 JSON 序列化为中国风格格式
type JSONTime time.Time

// MarshalJSON 实现 json.Marshaler 接口
func (t JSONTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

// String 实现 fmt.Stringer 接口
func (t JSONTime) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
}

// Format 转发 time.Time 的 Format 方法
func (t JSONTime) Format(layout string) string {
	return time.Time(t).Format(layout)
}

// Before 转发 time.Time 的 Before 方法
func (t JSONTime) Before(u JSONTime) bool {
	return time.Time(t).Before(time.Time(u))
}

// After 转发 time.Time 的 After 方法
func (t JSONTime) After(u JSONTime) bool {
	return time.Time(t).After(time.Time(u))
}

// IsZero 转发 time.Time 的 IsZero 方法
func (t JSONTime) IsZero() bool {
	return time.Time(t).IsZero()
}

// Unix 转发 time.Time 的 Unix 方法
func (t JSONTime) Unix() int64 {
	return time.Time(t).Unix()
}

// Time 返回底层的 time.Time 类型
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}

// Add 转发 time.Time 的 Add 方法，返回 JSONTime
func (t JSONTime) Add(d time.Duration) JSONTime {
	return JSONTime(time.Time(t).Add(d))
}

// Sub 转发 time.Time 的 Sub 方法
func (t JSONTime) Sub(u JSONTime) time.Duration {
	return time.Time(t).Sub(time.Time(u))
}

// Local 转发 time.Time 的 Local 方法
func (t JSONTime) Local() JSONTime {
	return JSONTime(time.Time(t).Local())
}

// UTC 转发 time.Time 的 UTC 方法
func (t JSONTime) UTC() JSONTime {
	return JSONTime(time.Time(t).UTC())
}

// Year 返回年份
func (t JSONTime) Year() int {
	return time.Time(t).Year()
}
