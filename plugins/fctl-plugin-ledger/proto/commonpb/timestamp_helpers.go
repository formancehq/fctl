package commonpb

import "time"

// AsTime converts a protobuf Timestamp to a Go time.Time.
func (x *Timestamp) AsTime() time.Time {
	if x == nil {
		return time.Time{}
	}
	return time.UnixMicro(int64(x.Data))
}
