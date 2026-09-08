-- Phase 24: Seed ACL permissions for Maintenance & Fleet models.
-- Group 1 (internal user): CRUD except delete. Group 2 (manager/admin): full CRUD.

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'maintenance.equipment', true, true, true, false),
    (2::BIGINT, 'maintenance.equipment', true, true, true, true),
    (1::BIGINT, 'maintenance.equipment.category', true, true, true, false),
    (2::BIGINT, 'maintenance.equipment.category', true, true, true, true),
    (1::BIGINT, 'maintenance.stage', true, false, false, false),
    (2::BIGINT, 'maintenance.stage', true, true, true, true),
    (1::BIGINT, 'maintenance.team', true, false, false, false),
    (2::BIGINT, 'maintenance.team', true, true, true, true),
    (1::BIGINT, 'maintenance.request', true, true, true, false),
    (2::BIGINT, 'maintenance.request', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model.brand', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model.brand', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model.category', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model.category', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.tag', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.tag', true, true, true, true),
    (1::BIGINT, 'fleet.service.type', true, true, true, false),
    (2::BIGINT, 'fleet.service.type', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.odometer', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.odometer', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.log.services', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.log.services', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.log.contract', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.log.contract', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;