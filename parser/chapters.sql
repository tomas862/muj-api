-- Junction table to map sections to chapters
CREATE TABLE section_chapter_mapping (
    section_number INT REFERENCES sections(section_number) ON DELETE CASCADE,
    chapter_id INT NOT NULL,
    PRIMARY KEY (section_number, chapter_id)
);

-- Map sections to chapters
INSERT INTO section_chapter_mapping (section_number, chapter_id) VALUES
-- Section I: Chapters 1-5
(1, 1), (1, 2), (1, 3), (1, 4), (1, 5),
-- Section II: Chapters 6-14
(2, 6), (2, 7), (2, 8), (2, 9), (2, 10),
(2, 11), (2, 12), (2, 13), (2, 14),
-- Section III: Chapters 15
(3, 15),
-- Section IV: Chapters 16-24
(4, 16), (4, 17), (4, 18), (4, 19), (4, 20),
(4, 21), (4, 22), (4, 23), (4, 24),
-- Section V: Chapters 25-27
(5, 25), (5, 26), (5, 27),
-- Section VI: Chapters 28-38
(6, 28), (6, 29), (6, 30), (6, 31), (6, 32),
(6, 33), (6, 34), (6, 35), (6, 36), (6, 37), (6, 38),
-- Section VII: Chapters 39-40
(7, 39), (7, 40),
-- Section VIII: Chapters 41-43
(8, 41), (8, 42), (8, 43),
-- Section IX: Chapters 44-46
(9, 44), (9, 45), (9, 46),
-- Section X: Chapters 47-49
(10, 47), (10, 48), (10, 49),
-- Section XI: Chapters 50-63
(11, 50), (11, 51), (11, 52), (11, 53), (11, 54),
(11, 55), (11, 56), (11, 57), (11, 58), (11, 59),
(11, 60), (11, 61), (11, 62), (11, 63),
-- Section XII: Chapters 64-67
(12, 64), (12, 65), (12, 66), (12, 67),
-- Section XIII: Chapters 68-70
(13, 68), (13, 69), (13, 70),
-- Section XIV: Chapters 71
(14, 71),
-- Section XV: Chapters 72-83
(15, 72), (15, 73), (15, 74), (15, 75), (15, 76),
(15, 77), (15, 78), (15, 79), (15, 80), (15, 81),
(15, 82), (15, 83),
-- Section XVI: Chapters 84-85
(16, 84), (16, 85),
-- Section XVII: Chapters 86-89
(17, 86), (17, 87), (17, 88), (17, 89),
-- Section XVIII: Chapters 90-92
(18, 90), (18, 91), (18, 92),
-- Section XIX: Chapters 93
(19, 93),
-- Section XX: Chapters 94-96
(20, 94), (20, 95), (20, 96),
-- Section XXI: Chapters 97-99
(21, 97), (21, 98), (21, 99);
