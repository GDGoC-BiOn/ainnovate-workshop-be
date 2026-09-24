-- seed.sql
-- Predictable IDs so workshop participants can use them directly.
--
--   user   -> 11111111-1111-1111-1111-111111111111 (Alice)
--   lesson -> 22222222-2222-2222-2222-222222222222 (Understand Hash Function)

INSERT INTO users (id, name)
VALUES ('11111111-1111-1111-1111-111111111111', 'Alice')
ON CONFLICT (id) DO NOTHING;

INSERT INTO lessons (id, title, objective)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Understand Hash Function',
    'Understand how hash functions map input data to a fixed-size value.'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO masteries (user_id, lesson_id, score)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    40
)
ON CONFLICT (user_id, lesson_id) DO UPDATE SET score = EXCLUDED.score;