package mapper

import (
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

func formatPort(p types.ServicePortConfig) string {
	var b strings.Builder
	if p.HostIP != "" {
		b.WriteString(p.HostIP)
		b.WriteByte(':')
	}
	if p.Published != "" {
		b.WriteString(p.Published)
		b.WriteByte(':')
	}
	b.WriteString(strconv.FormatUint(uint64(p.Target), 10))
	if p.Protocol != "" && p.Protocol != "tcp" {
		b.WriteByte('/')
		b.WriteString(p.Protocol)
	}
	return b.String()
}
