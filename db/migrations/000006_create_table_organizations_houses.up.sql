CREATE TABLE IF NOT EXISTS organizations_houses(
    organization_id INT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE ON UPDATE CASCADE,
    house_id INT NOT NULL REFERENCES houses(id) ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY (organization_id, house_id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);