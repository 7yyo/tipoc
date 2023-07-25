CREATE TABLE ${TABLE_NAME} (id INT, name VARCHAR(11), age INT);
INSERT INTO ${TABLE_NAME} VALUES(1, 'Green', 18), (2, 'Jim', 24);
CREATE TABLE ${TABLE_NAME}_order (user_id INT, order_id INT);
INSERT INTO ${TABLE_NAME}_order VALUES (1, 0001), (1, 0002), (2, 0003);
UPDATE ${TABLE_NAME} SET age = 24 WHERE id = 1;
UPDATE ${TABLE_NAME} SET age = age + 6 WHERE id = 2;
UPDATE ${TABLE_NAME} t1 JOIN ${TABLE_NAME}_order t2 ON t1.id = t2.user_id SET age = 88 WHERE t1.id = 1;
SELECT * FROM ${TABLE_NAME} t1 LEFT JOIN ${TABLE_NAME}_order t2 ON t1.id = t2.user_id;
