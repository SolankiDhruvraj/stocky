INSERT INTO stocks (symbol, name) VALUES ('RELIANCE', 'Reliance Industries') ON CONFLICT DO NOTHING;
INSERT INTO stocks (symbol, name) VALUES ('TCS', 'Tata Consultancy Services') ON CONFLICT DO NOTHING;
INSERT INTO stocks (symbol, name) VALUES ('INFY', 'Infosys') ON CONFLICT DO NOTHING;

INSERT INTO users (id, name) VALUES ('demo-user', 'Demo User') ON CONFLICT DO NOTHING;
