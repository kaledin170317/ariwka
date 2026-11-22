package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"vpn/internal/control/config/model"
	nodemodel "vpn/internal/control/node/model"
)

func (s Service) CreateDevice(ctx context.Context, userID, name string, defaultNetwork bool) (model.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return model.Device{}, errors.New("device name required (max 64 characters)")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	_, err := s.Users.Get(ctx, userID)
	if err != nil {
		return model.Device{}, err
	}

	device := model.Device{ID: uuid.NewString(), UserID: userID, Name: name, DefaultNetwork: defaultNetwork, Region: "RU"}

	err = s.Repo.CreateDevice(ctx, device)
	if err != nil {
		return model.Device{}, err
	}

	// The periodic sync fills in any node that is temporarily unavailable.
	return device, s.syncDevice(ctx, device)
}

func (s Service) ListDevices(ctx context.Context, userID string) ([]model.Device, error) {
	return s.Repo.ListDevices(ctx, userID)
}

func (s Service) DeleteDevice(ctx context.Context, userID, id string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	device, err := s.Repo.GetDevice(ctx, id)
	if err != nil {
		return err
	}

	if device.UserID != userID {
		return sql.ErrNoRows
	}

	profiles, err := s.Repo.ListByDevice(ctx, id)
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		err = s.removeProfile(ctx, profile)
		if err != nil {
			return fmt.Errorf("revoke %s: %w", profile.ID, err)
		}
	}

	return s.Repo.DeleteDevice(ctx, userID, id)
}

func (s Service) DeviceSubscription(ctx context.Context, id, hwid string) ([]string, bool, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	device, err := s.Repo.GetDevice(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if hwid == "" || device.HWID != "" && device.HWID != hwid {
		return nil, false, sql.ErrNoRows
	}

	err = s.syncDevice(ctx, device)
	if err != nil {
		return nil, false, err
	}

	profiles, err := s.Repo.ListByDevice(ctx, id)
	if err != nil {
		return nil, false, err
	}

	links := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		node, getErr := s.Nodes.Get(profile.NodeID)
		if getErr != nil {
			return nil, false, getErr
		}

		inbound, getErr := s.inbound(ctx, profile.NodeID, profile.Inbound)
		if getErr != nil {
			return nil, false, getErr
		}

		link, linkErr := node.Link(inbound, profile.ID)
		if linkErr != nil {
			return nil, false, linkErr
		}

		links = append(links, link)
	}

	bound, err := s.Repo.BindDevice(ctx, id, hwid)
	if err != nil {
		return nil, false, err
	}

	if !bound {
		return nil, false, sql.ErrNoRows
	}

	return links, device.DefaultNetwork, nil
}

// DeviceRouting describes only profiles belonging to this device. Call after
// DeviceSubscription has verified and bound the requesting HWID.
func (s Service) DeviceRouting(ctx context.Context, id string) (string, map[string]string, error) {
	device, err := s.Repo.GetDevice(ctx, id)
	if err != nil {
		return "", nil, err
	}
	profiles, err := s.Repo.ListByDevice(ctx, id)
	if err != nil {
		return "", nil, err
	}
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return "", nil, err
	}
	countries := make(map[string]string, len(nodes))
	for _, node := range nodes {
		countries[node.ID] = node.Country
	}
	byProfile := make(map[string]string, len(profiles))
	for _, profile := range profiles {
		country := countries[profile.NodeID]
		if country == "" {
			return "", nil, fmt.Errorf("missing country for node %s", profile.NodeID)
		}
		byProfile[profile.ID] = country
	}
	return device.Region, byProfile, nil
}

func (s Service) syncDevice(ctx context.Context, device model.Device) error {
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	profiles, err := s.Repo.ListByDevice(ctx, device.ID)
	if err != nil {
		return err
	}

	byInbound := make(map[string]model.Profile, len(profiles))
	for _, profile := range profiles {
		byInbound[profile.NodeID+":"+profile.Inbound] = profile
	}

	for _, n := range nodes {
		node, getErr := s.Nodes.Get(n.ID)
		if getErr != nil {
			return getErr
		}

		inbounds, getErr := s.Inbounds.Inbounds(ctx, n.ID)
		if getErr != nil {
			return getErr
		}

		for _, inbound := range inbounds {
			if device.DefaultNetwork {
				if inbound.Tag != "default:"+n.ID || inbound.Vless == nil {
					continue
				}
			} else if inbound.Network != nodemodel.InboundNetworkUsers {
				continue
			}

			profile, ok := byInbound[n.ID+":"+inbound.Tag]
			if !ok {
				profile = model.Profile{ID: uuid.NewString(), UserID: device.UserID,
					NodeID: n.ID, Inbound: inbound.Tag, DeviceID: device.ID}
				if inbound.Hysteria2 != nil {
					profile.Password = rand.Text()
				}

				getErr = s.Repo.Create(ctx, profile)
				if getErr != nil {
					return getErr
				}
			}

			getErr = s.syncProfile(ctx, node, inbound, profile)
			if getErr != nil {
				return fmt.Errorf("inbound %s: %w", inbound.Tag, getErr)
			}
		}
	}

	return nil
}

func (s Service) deleteProfile(ctx context.Context, userID, id string) error {
	profile, err := s.Repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}

	if profile.DeviceID != "" {
		return errors.New("delete the device to revoke its profiles")
	}

	return s.removeProfile(ctx, profile)
}

func (s Service) removeProfile(ctx context.Context, profile model.Profile) error {
	id := profile.ID

	node, err := s.Nodes.Get(profile.NodeID)
	if err != nil {
		return err
	}

	inbounds, err := s.Inbounds.Inbounds(ctx, profile.NodeID)
	if err != nil {
		return err
	}

	for _, inbound := range inbounds {
		if inbound.Tag == profile.Inbound {
			exists := false

			if h := inbound.Hysteria2; h != nil {
				for _, user := range h.Users {
					if user.Name == id {
						exists = true
						break
					}
				}
			} else if v := inbound.Vless; v != nil {
				for _, user := range v.UserDetails {
					if user.Tag == id {
						exists = true
						break
					}
				}

				if len(v.UserDetails) == 0 {
					exists = slices.Contains(v.Users, id)
				}
			}

			if exists {
				err = node.XrayRemoveUser(ctx, profile.Inbound, id)
				if err != nil {
					return err
				}
			}

			break
		}
	}

	return s.Repo.Delete(ctx, profile.UserID, id)
}
