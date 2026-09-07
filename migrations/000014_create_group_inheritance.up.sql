-- Group inheritance (implied groups) for effective RBAC membership.
CREATE TABLE IF NOT EXISTS res_groups_implied_rel (
    group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    implied_group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, implied_group_id),
    CONSTRAINT chk_res_groups_implied_not_self CHECK (group_id <> implied_group_id)
);

CREATE INDEX IF NOT EXISTS idx_res_groups_implied_rel_implied
    ON res_groups_implied_rel(implied_group_id);
