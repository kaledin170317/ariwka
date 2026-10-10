package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/config/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Create(ctx context.Context, p model.Profile) error {
	var deviceID any
	if p.DeviceID != "" {
		deviceID = p.DeviceID
	}

	_, err := builder.
		Insert("profiles").
		Columns("id", "user_id", "node_id", "inbound", "password", "device_id").
		Values(p.ID, p.UserID, p.NodeID, p.Inbound, p.Password, deviceID).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) Get(ctx context.Context, userID, id string) (model.Profile, error) {
	var p model.Profile

	err := builder.
		Select("id", "user_id", "node_id", "inbound", "password", "COALESCE(device_id, '')").
		From("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password, &p.DeviceID)

	return p, err
}

func (r Postgres) List(ctx context.Context, userID string) ([]model.Profile, error) {
	rows, err := builder.
		Select("id", "user_id", "node_id", "inbound", "password", "COALESCE(device_id, '')").
		From("profiles").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("node_id", "inbound", "id").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]model.Profile, 0)

	for rows.Next() {
		var p model.Profile

		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password, &p.DeviceID)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

func (r Postgres) ListByInbound(ctx context.Context, tag string) ([]model.Profile, error) {
	rows, err := builder.
		Select("id", "user_id", "node_id", "inbound", "password", "COALESCE(device_id, '')").
		From("profiles").
		Where(sq.Eq{"inbound": tag}).
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]model.Profile, 0)

	for rows.Next() {
		var p model.Profile

		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password, &p.DeviceID)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

func (r Postgres) ListByDevice(ctx context.Context, deviceID string) ([]model.Profile, error) {
	rows, err := builder.
		Select("id", "user_id", "node_id", "inbound", "password", "device_id").
		From("profiles").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("node_id", "inbound").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]model.Profile, 0)

	for rows.Next() {
		var p model.Profile

		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password, &p.DeviceID)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

func (r Postgres) CreateDevice(ctx context.Context, d model.Device) error {
	_, err := builder.Insert("devices").Columns("id", "user_id", "name").
		Values(d.ID, d.UserID, d.Name).RunWith(r.DB).ExecContext(ctx)

	return err
}

func (r Postgres) GetDevice(ctx context.Context, id string) (model.Device, error) {
	var d model.Device

	err := builder.Select("id", "user_id", "name", "COALESCE(hwid, '')").From("devices").Where(sq.Eq{"id": id}).
		RunWith(r.DB).QueryRowContext(ctx).Scan(&d.ID, &d.UserID, &d.Name, &d.HWID)

	return d, err
}

func (r Postgres) BindDevice(ctx context.Context, id, hwid string) (bool, error) {
	result, err := builder.Update("devices").Set("hwid", hwid).
		Where(sq.Eq{"id": id}).Where(sq.Or{sq.Eq{"hwid": nil}, sq.Eq{"hwid": hwid}}).
		RunWith(r.DB).ExecContext(ctx)
	if err != nil {
		return false, err
	}

	count, err := result.RowsAffected()

	return count != 0, err
}

func (r Postgres) ListDevices(ctx context.Context, userID string) ([]model.Device, error) {
	rows, err := builder.Select("id", "user_id", "name", "COALESCE(hwid, '')").From("devices").
		Where(sq.Eq{"user_id": userID}).OrderBy("name", "id").RunWith(r.DB).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]model.Device, 0)

	for rows.Next() {
		var d model.Device

		err = rows.Scan(&d.ID, &d.UserID, &d.Name, &d.HWID)
		if err != nil {
			return nil, err
		}

		devices = append(devices, d)
	}

	return devices, rows.Err()
}

func (r Postgres) DeleteDevice(ctx context.Context, userID, id string) error {
	result, err := builder.Delete("devices").Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).ExecContext(ctx)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r Postgres) SavePassword(ctx context.Context, id, password string) error {
	_, err := builder.
		Update("profiles").
		Set("password", password).
		Where(sq.Eq{"id": id, "password": ""}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) Delete(ctx context.Context, userID, id string) error {
	_, err := builder.
		Delete("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}
