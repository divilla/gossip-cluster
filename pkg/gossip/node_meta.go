package gossip

import "encoding/json"

type (
	NodeMeta struct {
		NodeID uint16
	}
)

func newNodeMetaBytes(id uint16) ([]byte, error) {
	nm := NodeMeta{NodeID: id}
	return json.Marshal(nm)
}
