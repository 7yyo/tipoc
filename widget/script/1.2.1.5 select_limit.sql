CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY , c1 INT, c2 INT);
INSERT INTO ${TABLE_NAME} VALUES (1, 10, 100), (2, 20, 200), (3, 30, 300);
SELECT * FROM ${TABLE_NAME} LIMIT 2;