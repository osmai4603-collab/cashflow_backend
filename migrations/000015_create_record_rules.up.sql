-- Typed record-rule definitions. The domain column stores validated AST JSON,
-- never executable SQL or Python expressions.
CREATE TABLE IF NOT EXISTS res_record_rules (
    id BIGSERIAL PRIMARY KEY,
    model VARCHAR(100) NOT NULL,
    group_id BIGINT REFERENCES res_groups(id) ON DELETE CASCADE,
    domain JSONB NOT NULL,
    can_read BOOLEAN NOT NULL DEFAULT false,
    can_create BOOLEAN NOT NULL DEFAULT false,
    can_update BOOLEAN NOT NULL DEFAULT false,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_res_record_rules_model ON res_record_rules(model);
CREATE INDEX IF NOT EXISTS idx_res_record_rules_group ON res_record_rules(group_id);
CREATE INDEX IF NOT EXISTS idx_res_record_rules_active ON res_record_rules(active);

CREATE TRIGGER trg_res_record_rules_updated_at
    BEFORE UPDATE ON res_record_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
