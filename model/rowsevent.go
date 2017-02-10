package model

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/funkygao/dbus/engine"
)

var (
	_ engine.Payloader = &RowsEvent{}
	_ sarama.Encoder   = &RowsEvent{}
)

// RowsEvent is a structured mysql binlog rows event.
// It implements engine.Payloader interface and can be transferred between plugins.
// It also implements kafka message value interface and can be produced to kafka.
type RowsEvent struct {
	Log       string `json:"log"`
	Position  uint32 `json:"pos"`
	Schema    string `json:"db"`
	Table     string `json:"tbl"`
	Action    string `json:"dml"`
	Timestamp uint32 `json:"ts"`

	// binlog has three update event version, v0, v1 and v2.
	// for v1 and v2, the rows number must be even.
	// Two rows for one event, format is [before update row, after update row]
	// for update v0, only one row for a event, and we don't support this version.
	Rows [][]interface{} `json:"rows"`

	bytes []byte
}

// Implements engine.Payloader.
func (r *RowsEvent) String() string {
	return fmt.Sprintf("%s %d %d %s %s/%s %+v", r.Log, r.Position, r.Timestamp, r.Action, r.Schema, r.Table, r.Rows)
}

func (r *RowsEvent) MetaInfo() string {
	return fmt.Sprintf("%s %d %d %s %s/%s", r.Log, r.Position, r.Timestamp, r.Action, r.Schema, r.Table)
}

// Implements engine.Payloader and sarama.Encoder.
func (r *RowsEvent) Encode() (b []byte, err error) {
	if len(r.bytes) > 0 {
		return r.bytes, nil
	}

	b, err = json.Marshal(r)
	r.bytes = b

	return
}

// Implements engine.Payloader and sarama.Encoder.
func (r *RowsEvent) Length() int {
	if len(r.bytes) == 0 {
		r.bytes, _ = json.Marshal(r)
	}

	return len(r.bytes)
}
