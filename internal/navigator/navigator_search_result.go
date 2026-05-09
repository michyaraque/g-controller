package navigator

import g "xabbo.b7c.io/goearth"

func Parse(packet *g.Packet) SearchResult {
	result := SearchResult{}

	result.SearchCode = packet.ReadString()
	result.FilteringData = packet.ReadString()

	blockCount := packet.ReadInt()
	result.Blocks = make([]Block, blockCount)

	for i := 0; i < blockCount; i++ {
		result.Blocks[i] = parseBlock(packet)
	}

	return result
}

func parseBlock(packet *g.Packet) Block {
	block := Block{}

	block.SearchCode = packet.ReadString()
	block.Text = packet.ReadString()
	block.ActionAllowed = packet.ReadInt()
	block.ForceClosed = packet.ReadBool()
	block.ViewMode = packet.ReadInt()

	roomCount := packet.ReadInt()
	block.Rooms = make([]Room, roomCount)
	for j := 0; j < roomCount; j++ {
		block.Rooms[j] = parseRoom(packet)
	}

	return block
}

func parseRoom(packet *g.Packet) Room {
	room := Room{}

	room.FlatID = packet.ReadInt()
	room.RoomName = packet.ReadString()
	room.OwnerID = packet.ReadInt()
	room.OwnerName = packet.ReadString()
	room.DoorMode = packet.ReadInt()
	room.UserCount = packet.ReadInt()
	room.MaxUserCount = packet.ReadInt()
	room.Description = packet.ReadString()
	room.TradeMode = packet.ReadInt()
	room.Score = packet.ReadInt()
	room.Ranking = packet.ReadInt()
	room.CategoryID = packet.ReadInt()

	tagCount := packet.ReadInt()
	room.Tags = make([]string, tagCount)
	for i := 0; i < tagCount; i++ {
		room.Tags[i] = packet.ReadString()
	}

	multiUse := packet.ReadInt()

	if (multiUse & 1) > 0 {
		picRef := packet.ReadString()
		room.OfficialPicRef = &picRef
	}

	if (multiUse & 2) > 0 {
		groupID := packet.ReadInt()
		groupName := packet.ReadString()
		badgeCode := packet.ReadString()
		room.GroupID = &groupID
		room.GroupName = &groupName
		room.GroupBadgeCode = &badgeCode
	}

	if (multiUse & 4) > 0 {
		adName := packet.ReadString()
		adDesc := packet.ReadString()
		adExp := packet.ReadInt()
		room.RoomAdName = &adName
		room.RoomAdDesc = &adDesc
		room.RoomAdExpires = &adExp
	}

	room.ShowOwner = (multiUse & 8) > 0
	room.AllowPets = (multiUse & 16) > 0
	room.DisplayEntryAd = (multiUse & 32) > 0

	return room
}

func CountRooms(result SearchResult) int {
	n := 0
	for _, b := range result.Blocks {
		n += len(b.Rooms)
	}
	return n
}
