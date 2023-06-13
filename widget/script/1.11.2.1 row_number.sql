CREATE TABLE ${TABLE_NAME} (id INT, name VARCHAR(11), age INT);
INSERT INTO ${TABLE_NAME} VALUES
                              (1, 'John', 25),
                              (2, 'Jane', 30),
                              (3, 'David', 28),
                              (4, 'Emma', 32),
                              (5, 'Michael', 27);
SELECT id, name , age , ROW_NUMBER() OVER (PARTITION BY name ORDER BY age) AS rn FROM ${TABLE_NAME};