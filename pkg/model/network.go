package model

type AddressIPVersionType string

const (
	IPv4 = AddressIPVersionType("ipv4")
	IPv6 = AddressIPVersionType("ipv6")
)

type AddressType string

const (
	InternetAddressType = AddressType("internet")
	IntranetAddressType = AddressType("intranet")
)

const (
	ECSBackendType = "ecs"
	ENIBackendType = "eni"
)

const (
	HTTP  = "http"
	HTTPS = "https"
	TCP   = "tcp"
	UDP   = "udp"
)

const (
	OnFlag  = FlagType("on")
	OffFlag = FlagType("off")
)

type FlagType string

var DEFAULT_PREFIX = "k8s"

type ModificationProtectionType string

const ConsoleProtection = ModificationProtectionType("ConsoleProtection")

const S1Small = "slb.s1.small"
