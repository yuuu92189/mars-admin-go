-- +goose Up
UPDATE access_menus
SET route_path = 'profile',
    component = 'profile/index'
WHERE route_path IN ('settings/profile', 'system/profile', 'profile/index');

-- +goose Down
UPDATE access_menus
SET route_path = 'settings/profile',
    component = 'profile/index'
WHERE route_path = 'profile'
  AND component = 'profile/index';
