// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ssntp/uuid"
)

func (c *controller) makePoolLinks(pool *types.Pool) {
	for i := range pool.Subnets {
		subnet := &pool.Subnets[i]

		ref := fmt.Sprintf("%s/pools/%s/subnets/%s",
			c.apiURL, pool.ID, subnet.ID)

		link := types.Link{
			Rel:  "self",
			Href: ref,
		}

		subnet.Links = []types.Link{link}
	}

	for i := range pool.IPs {
		IP := &pool.IPs[i]

		ref := fmt.Sprintf("%s/pools/%s/external-ips/%s",
			c.apiURL, pool.ID, IP.ID)

		link := types.Link{
			Rel:  "self",
			Href: ref,
		}

		IP.Links = []types.Link{link}
	}

	selfRef := fmt.Sprintf("%s/pools/%s", c.apiURL, pool.ID)
	link := types.Link{
		Rel:  "self",
		Href: selfRef,
	}

	pool.Links = []types.Link{link}
}

func (c *controller) makeMappedIPLinks(IP *types.MappedIP, tenant *string) {
	var ref string

	if tenant != nil {
		ref = fmt.Sprintf("%s/%s/external-ips/%s",
			c.apiURL, *tenant, IP.ID)
	} else {
		ref = fmt.Sprintf("%s/external-ips/%s",
			c.apiURL, IP.ID)
	}

	selfLink := types.Link{
		Rel:  "self",
		Href: ref,
	}

	IP.Links = []types.Link{selfLink}

	if tenant == nil {
		poolRef := fmt.Sprintf("%s/pools/%s", c.apiURL, IP.PoolID)
		link := types.Link{
			Rel:  "pool",
			Href: poolRef,
		}
		IP.Links = append(IP.Links, link)
	}
}

func (c *controller) AddPool(name string, subnet *string, ips []string) (types.Pool, error) {
	pools, err := c.ds.GetPools()
	if err != nil {
		return types.Pool{}, err
	}

	for _, p := range pools {
		if p.Name == name {
			return types.Pool{}, types.ErrDuplicatePoolName
		}
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: name,
	}

	if subnet != nil {
		sub := types.ExternalSubnet{
			ID:   uuid.Generate().String(),
			CIDR: *subnet,
		}

		_, ipNet, err := net.ParseCIDR(*subnet)
		if err != nil {
			return pool, err
		}

		ones, bits := ipNet.Mask.Size()

		// subtract out gateway and broadcast
		TotalIPs := (1 << uint32(bits-ones)) - 2
		pool.TotalIPs = TotalIPs
		pool.Free = pool.TotalIPs
		pool.Subnets = append(pool.Subnets, sub)
	} else if len(ips) > 0 {
		for _, i := range ips {
			addr := net.ParseIP(i)
			if addr == nil {
				return pool, types.ErrInvalidIP
			}

			IP := types.ExternalIP{
				ID:      uuid.Generate().String(),
				Address: i,
			}

			pool.IPs = append(pool.IPs, IP)
		}
		pool.TotalIPs = len(ips)
		pool.Free = pool.TotalIPs
	}

	err = c.ds.AddPool(pool)

	return pool, err
}

func (c *controller) ListPools() ([]types.Pool, error) {
	pools, err := c.ds.GetPools()
	if err != nil {
		return pools, err
	}

	// update the links. we do this here because we get the
	// current hostname:port.
	for i := range pools {
		pool := &pools[i]
		c.makePoolLinks(pool)
	}

	return pools, nil
}

func (c *controller) ShowPool(ID string) (types.Pool, error) {
	pool, err := c.ds.GetPool(ID)
	if err != nil {
		return pool, err
	}

	c.makePoolLinks(&pool)

	return pool, nil
}

func (c *controller) AddAddress(poolID string, subnet *string, ips []string) error {
	if subnet != nil {
		return c.ds.AddExternalSubnet(poolID, *subnet)
	}

	return c.ds.AddExternalIPs(poolID, ips)
}

func (c *controller) DeletePool(ID string) error {
	return c.ds.DeletePool(ID)
}

func (c *controller) RemoveAddress(poolID string, subnetID *string, IPID *string) error {
	if subnetID != nil {
		return c.ds.DeleteSubnet(poolID, *subnetID)
	}

	if IPID != nil {
		return c.ds.DeleteExternalIP(poolID, *IPID)
	}

	return types.ErrBadRequest
}

func (c *controller) ListMappedAddresses(tenant *string) []types.MappedIP {
	IPs := c.ds.GetMappedIPs(tenant)

	for i := range IPs {
		IP := &IPs[i]
		c.makeMappedIPLinks(IP, tenant)
	}

	return IPs
}

func (c *controller) MapAddress(poolName *string, instanceID string) error {
	var m types.MappedIP

	pools, err := c.ds.GetPools()
	if err != nil {
		return err
	}

	err = types.ErrPoolEmpty

	for _, pool := range pools {
		if poolName != nil {
			if pool.Name == *poolName {
				m, err = c.ds.MapExternalIP(pool.ID, instanceID)
				break
			}
		} else if pool.Free > 0 {
			m, err = c.ds.MapExternalIP(pool.ID, instanceID)
			break
		}
	}

	if err != nil {
		return err
	}

	// get tenant CNCI info
	t, err := c.ds.GetTenant(m.TenantID)
	if err != nil {
		_ = c.UnMapAddress(m.ExternalIP)
		return err
	}

	err = c.client.mapExternalIP(*t, m)
	if err != nil {
		// can never fail at this point.
		_ = c.UnMapAddress(m.ExternalIP)
	}

	return err
}

func (c *controller) UnMapAddress(address string) error {
	// get mapping
	m, err := c.ds.GetMappedIP(address)
	if err != nil {
		return err
	}

	// get tenant CNCI info
	t, err := c.ds.GetTenant(m.TenantID)
	if err != nil {
		return err
	}

	return c.client.unMapExternalIP(*t, m)
}
