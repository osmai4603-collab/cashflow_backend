-- Phase 24: Remove seeded ACL permissions for Maintenance & Fleet models.

DELETE FROM res_group_permissions
WHERE model IN (
    'maintenance.equipment',
    'maintenance.equipment.category',
    'maintenance.stage',
    'maintenance.team',
    'maintenance.request',
    'fleet.vehicle',
    'fleet.vehicle.model',
    'fleet.vehicle.model.brand',
    'fleet.vehicle.model.category',
    'fleet.vehicle.tag',
    'fleet.service.type',
    'fleet.vehicle.odometer',
    'fleet.vehicle.log.services',
    'fleet.vehicle.log.contract'
);