package libdynv6

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ZxwyProject/dynv6"
	"github.com/libdns/libdns"
)

const ttl = 60 * time.Second // default

var ErrUnsupportedType = errors.New(`unsupported record type`)

func recordToLibdns(r *dynv6.Record) libdns.Record {
	o := libdns.RR{
		Name: r.Name,
		TTL:  ttl,
		Type: r.Type,
	}
	switch r.Type {
	case dynv6.RT_A, dynv6.RT_AAAA, dynv6.RT_CNAME, dynv6.RT_TXT, dynv6.RT_SPF:
		// libdns.Address{}.RR()
		// libdns.CNAME{}.RR()
		// libdns.TXT{}.RR()
		o.Data = r.Data

	case dynv6.RT_CAA:
		// libdns.CAA{}.RR()
		if r.Flags != 0 || r.Tag != `` || r.Data != `` {
			o.Data = fmt.Sprintf(`%d %s %q`, r.Flags, r.Tag, r.Data)
		}

	case dynv6.RT_MX:
		// libdns.MX{}.RR()
		if r.Priority != 0 || r.Data != `` {
			o.Data = fmt.Sprintf("%d %s", r.Priority, r.Data)
		}

	case dynv6.RT_SRV:
		// libdns.SRV{}.RR()
		// TODO: Name?
		if r.Priority != 0 || r.Weight != 0 || r.Port != 0 || r.Data != `` {
			o.Data = fmt.Sprintf("%d %d %d %s", r.Priority, r.Weight, r.Port, r.Data)
		}

	default:
		// return nil
		panic(`unreachable`)
	}
	return &o
}

func recordFind(r []dynv6.Record, l *libdns.RR, n int) *dynv6.Record {
	for i := 0; i < n; i++ {
		a := &r[i]
		if a.Type == l.Type && a.Name == l.Name {
			return a
		}
	}
	return nil
}

func recordFromLibdns(l *libdns.RR) (*dynv6.RecordReq, error) {
	o := dynv6.RecordReq{
		Name: l.Name,
		Type: l.Type,
	}
	// l.Parse()
	switch l.Type {
	case dynv6.RT_A, dynv6.RT_AAAA, dynv6.RT_CNAME, dynv6.RT_TXT, dynv6.RT_SPF:
		o.Data = l.Data

	case dynv6.RT_CAA:
		fields := strings.Fields(l.Data)
		if expectedLen := 3; len(fields) != expectedLen {
			return nil, fmt.Errorf(`malformed CAA value; expected %d fields in the form 'flags tag "value"'`, expectedLen)
		}

		flags, err := strconv.ParseUint(fields[0], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid flags %s: %v", fields[0], err)
		}

		o.Flags = uint8(flags)
		o.Tag = fields[1]
		o.Data = strings.Trim(fields[2], `"`)

	case dynv6.RT_MX:
		fields := strings.Fields(l.Data)
		if expectedLen := 2; len(fields) != expectedLen {
			return nil, fmt.Errorf("malformed MX value; expected %d fields in the form 'preference target'", expectedLen)
		}

		priority, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid priority %s: %v", fields[0], err)
		}

		o.Priority = uint16(priority)
		o.Data = fields[1]

	case dynv6.RT_SRV:
		fields := strings.Fields(l.Data)
		if expectedLen := 4; len(fields) != expectedLen {
			return nil, fmt.Errorf("malformed SRV value; expected %d fields in the form 'priority weight port target'", expectedLen)
		}

		priority, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid priority %s: %v", fields[0], err)
		}
		weight, err := strconv.ParseUint(fields[1], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid weight %s: %v", fields[0], err)
		}
		port, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port %s: %v", fields[0], err)
		}

		// parts := strings.SplitN(l.Name, ".", 3)
		// if len(parts) < 2 {
		// 	return nil, fmt.Errorf("name %v does not contain enough fields; expected format: '_service._proto.name' or '_service._proto'", r.Name)
		// }
		// name := "@"
		// if len(parts) == 3 {
		// 	name = parts[2]
		// }
		// TODO: Name?

		o.Priority = uint16(priority)
		o.Weight = uint16(weight)
		o.Port = uint16(port)
		o.Data = fields[3]

	default:
		return nil, ErrUnsupportedType
	}
	return &o, nil
}
