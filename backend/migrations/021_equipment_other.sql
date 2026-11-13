-- Оборудование, которого нет в каталоге отдельной позицией (зарядка, док-станция, камера),
-- заказывается одной заявкой «Другое оборудование»: что именно – в комментарии.
INSERT INTO catalog_items (code, department_code, title, request_title, hint) VALUES
    ('other_equipment', 'equipment', 'Другое оборудование', 'Выдача оборудования',
     'Напиши в комментарии, что именно нужно и зачем.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO catalog_aliases (alias, item_code) VALUES
    ('зарядк', 'other_equipment'), ('блок питания', 'other_equipment'), ('адаптер', 'other_equipment'),
    ('переходник', 'other_equipment'), ('док-станц', 'other_equipment'), ('докстанц', 'other_equipment'),
    ('веб-камер', 'other_equipment'), ('вебкамер', 'other_equipment'), ('камер', 'other_equipment'),
    ('кабел', 'other_equipment'), ('подставк', 'other_equipment'), ('коврик', 'other_equipment'),
    ('зарядка для ноутбук', 'other_equipment'), ('зарядку для ноутбук', 'other_equipment'),
    ('зарядное', 'other_equipment'), ('второй монитор', 'monitor'), ('компьютерная мыш', 'peripherals')
ON CONFLICT DO NOTHING;
