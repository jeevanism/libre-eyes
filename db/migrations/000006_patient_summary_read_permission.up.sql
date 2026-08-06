INSERT INTO permissions (name, description)
VALUES ('patient.summary.read', 'Read the selected patient identity summary header')
ON CONFLICT (name) DO NOTHING;
