ALTER TABLE rewards DROP CONSTRAINT rewards_user_id_fkey;
ALTER TABLE holdings DROP CONSTRAINT holdings_user_id_fkey;

ALTER TABLE users ALTER COLUMN id TYPE TEXT;

ALTER TABLE rewards ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE holdings ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE daily_valuations ALTER COLUMN user_id TYPE TEXT;

ALTER TABLE rewards ADD CONSTRAINT rewards_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE holdings ADD CONSTRAINT holdings_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
