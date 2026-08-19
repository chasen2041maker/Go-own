package domain

type SecurityStatus string

const (
	SecurityStatusActive   SecurityStatus = "active"
	SecurityStatusInactive SecurityStatus = "inactive"
)

// Security 是可公开的静态目录项，不承载价格、涨跌或其他真实行情。
type Security struct {
	ID       int64
	Code     string
	Name     string
	Exchange string
	Status   SecurityStatus
}

// SecurityCursor 是仓储可理解的稳定位置；不透明与防篡改由 HTTP 边界负责。
type SecurityCursor struct {
	Code string
	ID   int64
}

type SecurityListQuery struct {
	Query    string
	Exchange string
	After    *SecurityCursor
	Limit    int
}
