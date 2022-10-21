package model

type AddressIPVersionType string

const (
	IPv4 = AddressIPVersionType("ipv4")
	IPv6 = AddressIPVersionType("ipv6")
)

const (
	ECSBackendType = "ecs"
	ENIBackendType = "eni"
)

const (
	OnFlag  = FlagType("on")
	OffFlag = FlagType("off")
)

type FlagType string
