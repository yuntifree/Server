package util

const (
	VerifyServerName   = "service:verify"
	VerifyServerPort   = ":50052"
	HotServerName      = "service:hot"
	HotServerPort      = ":50053"
	DiscoverServerName = "service:discover"
	DiscoverServerPort = ":50054"
	FetchServerName    = "service:fetch"
	FetchServerPort    = ":50055"
	ModifyServerName   = "service:modify"
	ModifyServerPort   = ":50056"
	PushServerName     = "service:push"
	PushServerPort     = ":50057"
	PunchServerName    = "service:punch"
	PunchServerPort    = ":50058"
	MaxIdleConns       = 3
	DebugHost          = "10.26.210.175"
	APIHosts           = "10.27.178.90,10.27.178.90"
	TimeFormat         = "2006-01-02 15:04:05"
	UidShareType       = 1
	GidShareType       = 2
	ListShareType      = 3
	TopShareType       = 4
	UserBetType        = 0
	UserAwardType      = 1
	InitStatus         = 0
	RunningStatus      = 1
	EndStatus          = 2
	AwardStatus        = 3
	AddressStatus      = 4
	ExpressStatus      = 5
	ReceiptStatus      = 6
	FiniStatus         = 7
	DiscoverServerType = 1
	VerifyServerType   = 2
	HotServerType      = 3
	FetchServerType    = 4
	ModifyServerType   = 5
	PushServerType     = 6
	PunchServerType    = 7
	AndroidTerm        = 0
	IosTerm            = 1
	WebTerm            = 2
	LoginType          = 0
	PortalType         = 1
)
