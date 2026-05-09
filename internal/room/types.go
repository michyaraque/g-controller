package room

import "time"

type Entity struct {
	Name     string
	JoinTime time.Time
	Tile     Tile
}

type Tile struct {
	X int
	Y int
}
