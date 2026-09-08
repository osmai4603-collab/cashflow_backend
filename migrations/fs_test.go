package migrations

import (
	"strings"
	"testing"
)

// TestFSBalancedUpDown ensures every .up.sql has a matching .down.sql and vice
// versa, so the migrator can always roll back a migration.
func TestFSBalancedUpDown(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read embedded migrations: %v", err)
	}

	var ups, downs []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			ups = append(ups, strings.TrimSuffix(name, ".up.sql"))
		case strings.HasSuffix(name, ".down.sql"):
			downs = append(downs, strings.TrimSuffix(name, ".down.sql"))
		}
	}

	if len(ups) != len(downs) {
		t.Fatalf("got %d .up.sql and %d .down.sql; each migration needs a pair", len(ups), len(downs))
	}
	for _, base := range ups {
		found := false
		for _, down := range downs {
			if down == base {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("migration %s is missing its .down.sql pair", base)
		}
	}
}

// TestFSLoyaltySchemaComplete verifies the loyalty migration defines the new
// tables and the sale merge columns.
func TestFSLoyaltySchemaComplete(t *testing.T) {
	up, err := FS.ReadFile("000040_create_loyalty_schema.up.sql")
	if err != nil {
		t.Fatalf("missing up migration: %v", err)
	}
	upStr := string(up)

	for _, table := range []string{
		"loyalty_programs",
		"loyalty_program_pricelists",
		"loyalty_rules",
		"loyalty_rewards",
		"loyalty_cards",
		"loyalty_card_history",
		"loyalty_mails",
		"sale_order_coupon_points",
	} {
		if !strings.Contains(upStr, "CREATE TABLE "+table) &&
			!strings.Contains(upStr, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("up migration is missing table %s", table)
		}
	}

	for _, col := range []string{
		"applied_coupon_ids",
		"code_enabled_rule_ids",
	} {
		if !strings.Contains(upStr, col) {
			t.Errorf("up migration is missing sale_orders column %s", col)
		}
	}

	for _, col := range []string{
		"reward_id",
		"coupon_id",
		"reward_identifier_code",
		"points_cost",
		"is_reward_line",
	} {
		if !strings.Contains(upStr, col) {
			t.Errorf("up migration is missing sale_order_lines column %s", col)
		}
	}

	down, err := FS.ReadFile("000040_create_loyalty_schema.down.sql")
	if err != nil {
		t.Fatalf("missing down migration: %v", err)
	}
	downStr := string(down)
	for _, table := range []string{
		"loyalty_programs",
		"loyalty_program_pricelists",
		"loyalty_rules",
		"loyalty_rewards",
		"loyalty_cards",
		"loyalty_card_history",
		"loyalty_mails",
		"sale_order_coupon_points",
	} {
		if !strings.Contains(downStr, "DROP TABLE IF EXISTS "+table) &&
			!strings.Contains(downStr, "DROP TABLE "+table) {
			t.Errorf("down migration is missing table %s", table)
		}
	}
}

// TestFSLoyaltyACLNamed matches the ACL migration file that seeds permissions.
func TestFSLoyaltyACLNamed(t *testing.T) {
	up, err := FS.ReadFile("000041_seed_loyalty_acl.up.sql")
	if err != nil {
		t.Fatalf("missing ACL migration: %v", err)
	}
	upStr := string(up)

	if !strings.Contains(upStr, "res_group_permissions") {
		t.Error("ACL migration must seed res_group_permissions")
	}

	for _, model := range []string{
		"loyalty.program",
		"loyalty.rule",
		"loyalty.reward",
		"loyalty.card",
		"loyalty.card.history",
		"loyalty.mail",
		"sale.order.coupon.points",
	} {
		if !strings.Contains(upStr, model) {
			t.Errorf("ACL seed is missing model %s", model)
		}
	}
}