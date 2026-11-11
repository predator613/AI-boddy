-- 014. «Новый диалог» в чате с помощником. Переписка не удаляется: сообщения до cleared_at
-- остаются в базе (метрики, разбор ошибок), но не показываются стажеру и не уходят модели.
ALTER TABLE conversations ADD COLUMN cleared_at TIMESTAMPTZ NULL;
