USE poc;
CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY, j JSON);
INSERT INTO ${TABLE_NAME} (id, j) VALUES (1, '{"name": "John", "age": 30, "city": "New York"}');
SELECT * FROM ${TABLE_NAME};
SELECT j ->> '$.name' FROM ${TABLE_NAME} WHERE id = 1;
