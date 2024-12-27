# Define variables for migration directory and SQLite database path
MIGRATION_DIR = ./migrations
DB_URL = "sqlite3://./data/database.db"

# Command to run migrations up
migrate-up:
	migrate -path $(MIGRATION_DIR) -database $(DB_URL) up

# Command to run migrations down
migrate-down:
	migrate -path $(MIGRATION_DIR) -database $(DB_URL) down

# Command to force a specific version
migrate-force:
	migrate -path $(MIGRATION_DIR) -database $(DB_URL) force $(version)

# Command to create a new migration file
migrate-create:
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $(name)