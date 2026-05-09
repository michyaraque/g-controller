package navigator

// SearchResult is the top-level container for a navigator search response.
type SearchResult struct {
	SearchCode    string     `json:"searchCode"`
	FilteringData string     `json:"filteringData"`
	Blocks        []Block    `json:"blocks"`
}

// Block groups rooms under a category or section.
type Block struct {
	SearchCode    string   `json:"searchCode"`
	Text          string   `json:"text"`
	ActionAllowed int      `json:"actionAllowed"`
	ForceClosed   bool     `json:"forceClosed"`
	ViewMode      int      `json:"viewMode"`
	Rooms         []Room   `json:"rooms"`
}

// Room represents a single room in the navigator result.
type Room struct {
	FlatID         int      `json:"flatId"`
	RoomName       string   `json:"roomName"`
	OwnerID        int      `json:"ownerId"`
	OwnerName      string   `json:"ownerName"`
	DoorMode       int      `json:"doorMode"`
	UserCount      int      `json:"userCount"`
	MaxUserCount   int      `json:"maxUserCount"`
	Description    string   `json:"description"`
	TradeMode      int      `json:"tradeMode"`
	Score          int      `json:"score"`
	Ranking        int      `json:"ranking"`
	CategoryID     int      `json:"categoryId"`
	Tags           []string `json:"tags"`
	OfficialPicRef *string  `json:"officialRoomPicRef,omitempty"`
	GroupID        *int     `json:"groupId,omitempty"`
	GroupName      *string  `json:"groupName,omitempty"`
	GroupBadgeCode *string  `json:"groupBadgeCode,omitempty"`
	RoomAdName     *string  `json:"roomAdName,omitempty"`
	RoomAdDesc     *string  `json:"roomAdDescription,omitempty"`
	RoomAdExpires  *int     `json:"roomAdExpiresInMin,omitempty"`
	ShowOwner      bool     `json:"showOwner"`
	AllowPets      bool     `json:"allowPets"`
	DisplayEntryAd bool     `json:"displayRoomEntryAd"`
}
