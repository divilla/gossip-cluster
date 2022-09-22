package gossip

type (
	SelectLeaderMessage struct {
		Method string `json:"method"`
		Args   struct {
			Leader uint16 `json:"leader"`
		} `json:"args"`
	}
)
