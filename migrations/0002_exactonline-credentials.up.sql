CREATE TABLE credential (
	recras_hostname TEXT PRIMARY KEY NOT NULL,
	recras_username TEXT NOT NULL,
	recras_password TEXT NOT NULL,
	exact_access_token TEXT NULL,
	exact_refresh_token TEXT NULL,
	state TEXT NOT NULL,
	start_sync_date DATE NOT NULL
);
