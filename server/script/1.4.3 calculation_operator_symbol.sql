CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY, num1 INT, num2 INT);
INSERT INTO ${TABLE_NAME} VALUES (1, 10, 5), (2, 20, 3), (3, 15, 8);
SELECT num1 + num2, num1 - num2, num1 * num2, num1 / num2, num1 % num2 FROM ${TABLE_NAME};
