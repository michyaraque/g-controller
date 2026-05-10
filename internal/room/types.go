package room

import "time"

type Entity struct {
	HabboID  int
	Name     string
	JoinTime time.Time
	Tile     Tile
	Dir      int
}

type Tile struct {
	X int
	Y int
}
