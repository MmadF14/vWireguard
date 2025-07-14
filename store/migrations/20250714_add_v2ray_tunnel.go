package migrations

import "database/sql"

type Migration struct {
	Name string
	Up   func(*sql.DB) error
	Down func(*sql.DB) error
}

var Migrations []Migration

func init() {
	Migrations = append(Migrations, Migration{
		Name: "20250714_add_v2ray_tunnel",
		Up: func(db *sql.DB) error {
			_, err := db.Exec(`ALTER TABLE tunnels ADD COLUMN v2ray_config JSON`)
			return err
		},
		Down: func(db *sql.DB) error {
			_, err := db.Exec(`ALTER TABLE tunnels DROP COLUMN v2ray_config`)
			return err
		},
	})
}
