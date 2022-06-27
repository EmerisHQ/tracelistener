package database

import "fmt"

// RunMigrations run all the migrations contained in "migrations" on the database pointed by dbConnString.
func RunMigrations(dbConnString string, migrations []string) error {
	c, err := New(dbConnString)
	if err != nil {
		return err
	}

	for i, m := range migrations {
		_, err := c.DB.Exec(m)
		if err != nil {
			return fmt.Errorf("error while running migration #%d, %w", i, err)
		}
	}

	return c.DB.Close()
}
