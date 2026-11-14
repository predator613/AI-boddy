-- Логины тестовых пользователей вслед за именами (миграция 020): Мария и Артемий.
-- Связи в базе идут по id, поэтому меняется только строка входа.
UPDATE users SET login = 'intern_maria' WHERE login = 'intern_anna';
UPDATE users SET login = 'lead_artemiy' WHERE login = 'lead_oleg';
