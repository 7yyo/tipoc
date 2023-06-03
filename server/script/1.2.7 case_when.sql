CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY, value INT);
INSERT INTO ${TABLE_NAME} VALUES (1, 10), (2, 20), (3, 30), (4, 40), (5, 50);
SELECT
    id,
    value,
    CASE
        WHEN value < 20 THEN '< 20'
        WHEN value >= 20 AND value < 40 THEN '>=20 and <=40'
        ELSE '>=40'
        END AS category
FROM
    ${TABLE_NAME};
