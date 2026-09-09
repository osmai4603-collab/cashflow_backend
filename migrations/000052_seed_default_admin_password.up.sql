-- Ensure the development administrator can sign in after database initialization.
UPDATE res_users
SET password_hash = '$2a$10$CM3u7Nj5os9FgJqaBfnNoe.0y64Uu.eR9H94FQxWu/XLHLLL9NNJ2',
    active = TRUE,
    updated_at = NOW()
WHERE login = 'admin';