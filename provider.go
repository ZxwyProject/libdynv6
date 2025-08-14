// Package libdnstemplate implements a DNS record management client compatible
// with the libdns interfaces for Dynv6 REST API.
package libdynv6

import (
	"context"
	"sync"

	"github.com/ZxwyProject/dynv6"
	"github.com/libdns/libdns"
)

// TODO: Providers must not require additional provisioning steps by the callers; it
// should work simply by populating a struct and calling methods on it. If your DNS
// service requires long-lived state or some extra provisioning step, do it implicitly
// when methods are called; sync.Once can help with this, and/or you can use a
// sync.(RW)Mutex in your Provider struct to synchronize implicit provisioning.

// Provider facilitates DNS record manipulation with Dynv6 REST API.
type Provider struct {
	o sync.Once // for init

	Dynv6 *dynv6.Client `json:"-"` // internal client

	//# HTTP Token
	//
	// You can get it at https://dynv6.com/keys
	Token string `json:"token,omitempty"`

	// TODO: Put config fields here (with snake_case json struct tags on exported fields), for example:
	// Exported config fields should be JSON-serializable or omitted (`json:"-"`)
}

func (p *Provider) init() {
	// You must ensure that the token is filled in before the first call!
	if p.Token == `` {
		panic(`libdynv6: No token provided!`)
	}
	p.Dynv6 = dynv6.NewClient(p.Token)
}

// GetRecords returns all the records in the DNS zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.o.Do(p.init)
	z, err := p.Dynv6.ZoneNameCtx(ctx, zone)
	if err != nil {
		return nil, err
	}
	r, err := p.Dynv6.RecordsCtx(ctx, string(z.ID))
	if err != nil {
		return nil, err
	}
	l := len(r)
	o := make([]libdns.Record, l)

	for i := 0; i < l; i++ {
		o[i] = recordToLibdns(&r[i])
	}
	// Make sure to return RR-type-specific structs, not libdns.RR structs.
	return o, nil
}

// AppendRecords creates the inputted records in the given zone and returns the populated records that were created.
// It never changes existing records.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.o.Do(p.init)
	z, err := p.Dynv6.ZoneNameCtx(ctx, zone)
	if err != nil {
		return nil, err
	}
	r, err := p.Dynv6.RecordsCtx(ctx, string(z.ID))
	if err != nil {
		return nil, err
	}
	l, m, n := len(records), len(r), 0
	o := make([]libdns.Record, l)

	for i := 0; i < l; i++ {
		li := records[i]
		lr := li.RR()

		if recordFind(r, &lr, m) != nil {
			if dynv6.Debug {
				dynv6.DbgLog.Println(`[Dynv6-debug/libdns] AppendRecords:`, libdns.AbsoluteName(lr.Name, zone), `already exists!`)
			}
			continue
		}

		dr, err := recordFromLibdns(&lr)
		if err != nil {
			return nil, err
		}

		_, err = p.Dynv6.RecordAddCtx(ctx, string(z.ID), dr)
		if err != nil {
			return nil, err
		}
		o[n] = lr
		n++
	}
	// Make sure to return RR-type-specific structs, not libdns.RR structs.
	return o[:n], nil
}

// SetRecords updates the zone so that the records described in the input are reflected in the output.
// It may create or update records or—depending on the record type—delete records to maintain parity with the input.
// No other records are affected. It returns the records which were set.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.o.Do(p.init)
	z, err := p.Dynv6.ZoneNameCtx(ctx, zone)
	if err != nil {
		return nil, err
	}
	r, err := p.Dynv6.RecordsCtx(ctx, string(z.ID))
	if err != nil {
		return nil, err
	}
	l, m := len(records), len(r)
	o := make([]libdns.Record, l)

	for i := 0; i < l; i++ {
		li := records[i]
		lr := li.RR()

		dr, err := recordFromLibdns(&lr)
		if err != nil {
			return nil, err
		}

		fr := recordFind(r, &lr, m)
		if fr == nil {
			// new
			_, err = p.Dynv6.RecordAddCtx(ctx, string(z.ID), dr)
		} else {
			// upd
			_, err = p.Dynv6.RecordUpdCtx(ctx, string(z.ID), string(fr.ID), dr)
		}
		if err != nil {
			return nil, err
		}
		o[i] = lr
	}
	// Make sure to return RR-type-specific structs, not libdns.RR structs.
	return o, nil
}

// DeleteRecords deletes the given records from the zone if they exist in the zone and exactly match the input.
// If the input records do not exist in the zone, they are silently ignored.
// DeleteRecords returns only the the records that were deleted, and does not return any records that were provided in the input but did not exist in the zone.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.o.Do(p.init)
	z, err := p.Dynv6.ZoneNameCtx(ctx, zone)
	if err != nil {
		return nil, err
	}
	r, err := p.Dynv6.RecordsCtx(ctx, string(z.ID))
	if err != nil {
		return nil, err
	}
	l, m, n := len(records), len(r), 0
	o := make([]libdns.Record, l)

	for i := 0; i < l; i++ {
		li := records[i]
		lr := li.RR()

		fr := recordFind(r, &lr, m)
		if fr == nil {
			continue
		}

		err = p.Dynv6.RecordDelCtx(ctx, string(z.ID), string(fr.ID))
		if err != nil {
			return nil, err
		}
		o[n] = lr
		n++
	}
	// Make sure to return RR-type-specific structs, not libdns.RR structs.
	return o[:n], nil
}

// ListZones returns the list of available DNS zones for use by other [libdns] methods.
func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	p.o.Do(p.init)
	z, err := p.Dynv6.ZonesCtx(ctx)
	if err != nil {
		return nil, err
	}
	l := len(z)
	o := make([]libdns.Zone, l)

	for i := 0; i < l; i++ {
		o[i] = libdns.Zone{
			Name: z[i].Name,
		}
	}
	return o, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
	_ libdns.ZoneLister     = (*Provider)(nil)
)
