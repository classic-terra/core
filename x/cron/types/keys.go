package types

const (
	// ModuleName defines the module name.
	ModuleName = "cron"
	// StoreKey defines the primary module store key.
	StoreKey = ModuleName
	// RouterKey defines the module message route.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_cron"
)

const (
	prefixScheduleKey = iota + 1
	prefixScheduleCountKey
	prefixParamsKey
)

var (
	ScheduleKey      = []byte{prefixScheduleKey}
	ScheduleCountKey = []byte{prefixScheduleCountKey}
	ParamsKey        = []byte{prefixParamsKey}
)

// GetScheduleKey returns the store key for a schedule name.
func GetScheduleKey(name string) []byte {
	return []byte(name)
}
